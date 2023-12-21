package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/ardanlabs/conf/v3"
	"github.com/graphql-go/graphql"
	"github.com/ldebruijn/go-graphql-armor/internal/app/config"
	"github.com/ldebruijn/go-graphql-armor/internal/business/aliases"
	"github.com/ldebruijn/go-graphql-armor/internal/business/block_field_suggestions"
	"github.com/ldebruijn/go-graphql-armor/internal/business/gql"
	middleware2 "github.com/ldebruijn/go-graphql-armor/internal/business/middleware"
	"github.com/ldebruijn/go-graphql-armor/internal/business/persisted_operations"
	"github.com/ldebruijn/go-graphql-armor/internal/business/proxy"
	"github.com/ldebruijn/go-graphql-armor/internal/business/readiness"
	"github.com/ldebruijn/go-graphql-armor/internal/business/schema"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log2 "log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

var (
	build       = "develop"
	configPath  = ""
	httpCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "go_graphql_armor",
		Subsystem: "http",
		Name:      "count",
		Help:      "HTTP request counts",
	},
		[]string{"route"},
	)
	httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "go_graphql_armor",
		Subsystem: "http",
		Name:      "duration",
		Help:      "HTTP duration",
	},
		[]string{"route"},
	)
)

func init() {
	prometheus.MustRegister(httpCounter, httpDuration)
}

func main() {
	flag.StringVar(&configPath, "f", "./armor.yml", "Defines the path at which the configuration file can be found")
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

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	if err := run(log, cfg, shutdown); err != nil {
		log.Error("startup", "msg", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger, cfg *config.Config, shutdown chan os.Signal) error {
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

	mid := middleware(log, cfg, po, schemaProvider)

	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/internal/healthz/readiness", readiness.NewReadinessHandler())
	mux.Handle(cfg.Web.Path, mid(Handler(pxy)))

	api := http.Server{
		Addr:         cfg.Web.Host,
		Handler:      mux,
		ReadTimeout:  cfg.Web.ReadTimeout,
		WriteTimeout: cfg.Web.WriteTimeout,
		IdleTimeout:  cfg.Web.IdleTimeout,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Info("startup", "status", "go-graphql-armor started", "host", api.Addr)

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

func middleware(log *slog.Logger, cfg *config.Config, po *persisted_operations.PersistedOperationsHandler, schema *schema.Provider) func(next http.Handler) http.Handler {
	rec := middleware2.Recover(log)
	httpInstrumentation := HttpInstrumentation()

	// clear validation rules as we leave operartion validation to the actual backend
	graphql.SpecifiedRules = []graphql.ValidationRuleFn{}

	_ = aliases.NewMaxAliasesRule(cfg.MaxAliases)
	vr := ValidationRules(schema)

	fn := func(next http.Handler) http.Handler {
		return rec(httpInstrumentation(po.Execute(vr(next))))
	}

	return fn
}

func HttpInstrumentation() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			httpCounter.WithLabelValues(r.URL.Path).Inc()

			next.ServeHTTP(w, r)

			httpDuration.WithLabelValues(r.URL.Path).Observe(time.Since(start).Seconds())
		}
		return http.HandlerFunc(fn)
	}
}

func ValidationRules(schema *schema.Provider) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			payload, err := gql.ParseRequestPayload(r)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			//sm := schema.Get()

			params := graphql.Params{
				RequestString: payload.Query,
				Context:       r.Context(),
				//Schema:        schema.Get(),
			}
			result := graphql.Do(params)

			if result.HasErrors() {
				_ = json.NewEncoder(w).Encode(result)
				return
			}

			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

func Handler(p *httputil.ReverseProxy) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		p.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
