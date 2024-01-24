package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/ardanlabs/conf/v3"
	"github.com/ldebruijn/graphql-protect/internal/app/config"
	"github.com/ldebruijn/graphql-protect/internal/business/aliases"
	"github.com/ldebruijn/graphql-protect/internal/business/batch"
	"github.com/ldebruijn/graphql-protect/internal/business/block_field_suggestions"
	"github.com/ldebruijn/graphql-protect/internal/business/enforce_post"
	"github.com/ldebruijn/graphql-protect/internal/business/gql"
	"github.com/ldebruijn/graphql-protect/internal/business/max_depth"
	"github.com/ldebruijn/graphql-protect/internal/business/persisted_operations"
	"github.com/ldebruijn/graphql-protect/internal/business/schema"
	"github.com/ldebruijn/graphql-protect/internal/business/tokens"
	"github.com/ldebruijn/graphql-protect/internal/http/middleware"
	"github.com/ldebruijn/graphql-protect/internal/http/proxy"
	"github.com/ldebruijn/graphql-protect/internal/http/readiness"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/vektah/gqlparser/v2/parser"
	"github.com/vektah/gqlparser/v2/validator"
	log2 "log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

var (
	shortHash  = "develop"
	build      = "develop"
	configPath = ""

	appInfo = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "graphql_protect",
		Subsystem: "app",
		Name:      "info",
		Help:      "Application information",
	},
		[]string{"version", "go_version", "short_hash"},
	)
	errRedacted = errors.New("error(s) redacted")
)

func init() {
	prometheus.MustRegister(appInfo)
}

func main() {
	flag.StringVar(&configPath, "f", "./protect.yml", "Defines the path at which the configuration file can be found")
	flag.Parse()

	log := slog.Default()

	// cfg
	cfg, err := config.NewConfig(configPath)
	if err != nil {
		log.Error("Error loading application configuration", "err", err)
		os.Exit(1)
	}
	cfgAsString, _ := conf.String(cfg)
	log2.Println(cfgAsString)

	log.Info("Starting service", "version", build)

	appInfo.With(prometheus.Labels{
		"version":    build,
		"go_version": runtime.Version(),
		"short_hash": shortHash,
	})

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	if err := run(log, cfg, shutdown); err != nil {
		log.Error("startup", "msg", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger, cfg *config.Config, shutdown chan os.Signal) error { // nolint:funlen
	log.Info("startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))

	log.Info("Starting proxy", "target", cfg.Target.Host)

	blockFieldSuggestions := block_field_suggestions.NewBlockFieldSuggestionsHandler(cfg.BlockFieldSuggestions)

	pxy, err := proxy.NewProxy(cfg.Target, blockFieldSuggestions)
	if err != nil {
		log.Error("ErrorPayload creating proxy", "err", err)
		return nil
	}

	remoteLoader, err := persisted_operations.RemoteLoaderFromConfig(cfg.PersistedOperations)
	if err != nil && !errors.Is(err, persisted_operations.ErrNoRemoteLoaderSpecified) {
		log.Warn("Error initializing remote loader", "err", err)
	}

	po, err := persisted_operations.NewPersistedOperations(log, cfg.PersistedOperations, persisted_operations.NewLocalDirLoader(cfg.PersistedOperations), remoteLoader)
	if err != nil {
		log.Error("Error initializing Persisted Operations", "err", err)
		return nil
	}

	schemaProvider, err := schema.NewSchema(cfg.Schema, log)
	if err != nil {
		log.Error("Error initializing schema", "err", err)
		return nil
	}

	mux := http.NewServeMux()

	mid := middlewareChain(log, cfg, po, schemaProvider)

	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/internal/healthz/readiness", readiness.NewReadinessHandler())
	mux.Handle(cfg.Web.Path, mid(pxy))

	api := http.Server{
		Addr:         cfg.Web.Host,
		Handler:      mux,
		ReadTimeout:  cfg.Web.ReadTimeout,
		WriteTimeout: cfg.Web.WriteTimeout,
		IdleTimeout:  cfg.Web.IdleTimeout,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Info("startup", "status", "graphql-protect started", "host", api.Addr)

		serverErrors <- api.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		log.Info("shutdown", "status", "shutdown started", "signal", sig)
		defer log.Info("shutdown", "status", "shutdown complete", "signal", sig)

		ctx, cancel := context.WithTimeout(context.Background(), cfg.Web.ShutdownTimeout)
		defer cancel()

		po.Shutdown()

		if err := api.Shutdown(ctx); err != nil {
			_ = api.Close()
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}

	return nil
}

func middlewareChain(log *slog.Logger, cfg *config.Config, po *persisted_operations.PersistedOperationsHandler, schema *schema.Provider) func(next http.Handler) http.Handler {
	rec := middleware.Recover(log)
	httpInstrumentation := middleware.RequestMetricMiddleware()

	aliases.NewMaxAliasesRule(cfg.MaxAliases)
	max_depth.NewMaxDepthRule(cfg.MaxDepth)
	tks := tokens.MaxTokens(cfg.MaxTokens)
	maxBatch, err := batch.NewMaxBatch(cfg.MaxBatch)
	if err != nil {
		log.Warn("Error initializing maximum batch protection", err)
	}

	vr := ValidationRules(schema, tks, maxBatch, cfg.ObfuscateValidationErrors)
	disableMethod := enforce_post.EnforcePostMethod(cfg.EnforcePost)

	fn := func(next http.Handler) http.Handler {
		return rec(httpInstrumentation(disableMethod(po.Execute(vr(next)))))
	}

	return fn
}

func ValidationRules(schema *schema.Provider, tks *tokens.MaxTokensRule, batch *batch.MaxBatchRule, obfuscateErrors bool) func(next http.Handler) http.Handler { // nolint:funlen,cyclop
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			payload, err := gql.ParseRequestPayload(r)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			var errs gqlerror.List

			err = batch.Validate(payload)
			if err != nil {
				errs = append(errs, gqlerror.Wrap(err))
			}

			// only process the rest if no error yet
			if err == nil {
				for _, data := range payload {
					operationSource := &ast.Source{
						Input: data.Query,
					}

					err = tks.Validate(operationSource)
					if err != nil {
						errs = append(errs, gqlerror.Wrap(err))
						continue // we could consider break-ing here. That would short-circuit on error, with the downside of not returning all potential errors
					}

					var query, err = parser.ParseQuery(operationSource)
					if err != nil {
						errs = append(errs, gqlerror.Wrap(err))
						continue
					}

					errList := validator.Validate(schema.Get(), query)
					if len(errList) > 0 {
						errs = append(errs, errList...)
						continue
					}
				}
			}

			if len(errs) > 0 {
				if obfuscateErrors {
					errs = gqlerror.List{gqlerror.Wrap(errRedacted)}
				}

				response := map[string]interface{}{
					"data":   nil,
					"errors": errs,
				}

				err = json.NewEncoder(w).Encode(response)
				if err != nil {
					log2.Println(err)
				}
				return
			}

			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
