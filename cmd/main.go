package main

import (
	"context"
	"encoding/json"
	"errors"
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
	log2 "log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

var build = "develop"

func main() {
	log := slog.Default()

	// cfg
	cfg, err := config.NewConfig()
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

	mux := http.NewServeMux()

	mid := middleware(log, cfg, po)
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

func middleware(log *slog.Logger, cfg *config.Config, po *persisted_operations.PersistedOperationsHandler) func(next http.Handler) http.Handler {
	rec := middleware2.Recover(log)
	_ = aliases.NewMaxAliasesRule(cfg.MaxAliases)
	vr := ValidationRules()

	fn := func(next http.Handler) http.Handler {
		return rec(po.Execute(vr(next)))
	}

	return fn
}

func ValidationRules() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			payload, err := gql.ParseRequestPayload(r)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			params := graphql.Params{
				RequestString: payload.Query,
				Context:       r.Context(),
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
