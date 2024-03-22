package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/ardanlabs/conf/v3"
	"github.com/ldebruijn/graphql-protect/internal/app/config"
	"github.com/ldebruijn/graphql-protect/internal/app/otel"
	"github.com/ldebruijn/graphql-protect/internal/business/persisted_operations"
	"github.com/ldebruijn/graphql-protect/internal/business/protect"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/block_field_suggestions"
	"github.com/ldebruijn/graphql-protect/internal/business/schema"
	"github.com/ldebruijn/graphql-protect/internal/http/proxy"
	"github.com/ldebruijn/graphql-protect/internal/http/readiness"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	log.Info("Starting Protect", "version", build)

	appInfo.With(prometheus.Labels{
		"version":    build,
		"go_version": runtime.Version(),
		"short_hash": shortHash,
	})

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// serve as default to prevent breaking change
	action := "serve"
	if len(os.Args) > 1 {
		action = os.Args[1]
	}

	switch action {
	case "serve":
		if err := httpServer(log, cfg, shutdown); err != nil {
			log.Error("startup", "msg", err)
			os.Exit(1)
		}
	case "validate":
		if err := validate(log, cfg, shutdown); err != nil {
			log.Error("validate", "msg", err)
			os.Exit(1)
		}
	case "version":
		log.Info("GraphQL Protect", "version", build, "go_version", runtime.Version(), "short_hash", shortHash)
		os.Exit(0)
	default:
		log.Error("Subcommand required", "subcommand", action)
		os.Exit(1)
	}

}

func run(log *slog.Logger, cfg *config.Config, shutdown chan os.Signal) error { // nolint:funlen,cyclop
	log.Info("startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))

	shutDownTracer, err := otel.SetupOTELSDK(context.Background(), build)
	if err != nil {
		log.Error("Could not setup OTEL Tracing, continuing without tracing")
	}

	log.Info("Starting proxy", "target", cfg.Target.Host)

	blockFieldSuggestions := block_field_suggestions.NewBlockFieldSuggestionsHandler(cfg.BlockFieldSuggestions)

	pxy, err := proxy.NewProxy(cfg.Target, blockFieldSuggestions)
	if err != nil {
		log.Error("ErrorPayload creating proxy", "err", err)
		return nil
	}

	remoteLoader, err := persisted_operations.RemoteLoaderFromConfig(cfg.PersistedOperations, log)
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

	protectHandler, err := protect.NewGraphQLProtect(log, cfg, po, schemaProvider, pxy)
	if err != nil {
		log.Error("Error initializing GraphQL Protect", "err", err)
		return err
	}

	mux := http.NewServeMux()

	mid := protectMiddlewareChain(log)

	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/internal/healthz/readiness", readiness.NewReadinessHandler())
	mux.Handle(cfg.Web.Path, mid(protectHandler))

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
		if err := shutDownTracer(ctx); err != nil {
			log.Error("Could not shutdown tracing gracefully", "err", err)
		}
	}

	return nil
}
