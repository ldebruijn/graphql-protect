package main

import (
	"context"
	"fmt"
	"github.com/ldebruijn/graphql-protect/internal/app/config"
	_ "github.com/ldebruijn/graphql-protect/internal/app/metrics"
	"github.com/ldebruijn/graphql-protect/internal/app/otel"
	"github.com/ldebruijn/graphql-protect/internal/business/protect"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/block_field_suggestions"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/obfuscate_upstream_errors"
	"github.com/ldebruijn/graphql-protect/internal/business/schema"
	"github.com/ldebruijn/graphql-protect/internal/business/trusteddocuments"
	"github.com/ldebruijn/graphql-protect/internal/http/debug"
	"github.com/ldebruijn/graphql-protect/internal/http/middleware"
	"github.com/ldebruijn/graphql-protect/internal/http/proxy"
	"github.com/ldebruijn/graphql-protect/internal/http/readiness"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"log/slog"
	"net/http"
	"os"
	"runtime"
)

func httpServer(log *slog.Logger, cfg *config.Config, shutdown chan os.Signal) error { // nolint:funlen,cyclop
	log.Info("startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))

	shutDownTracer, err := otel.SetupOTELSDK(context.Background(), build)
	if err != nil {
		log.Error("Could not setup OTEL Tracing, continuing without tracing")
	}

	log.Info("Starting proxy", "target", cfg.Target.Host)

	blockFieldSuggestions := block_field_suggestions.NewBlockFieldSuggestionsHandler(cfg.BlockFieldSuggestions)
	obfuscateUpstreamErrors := obfuscate_upstream_errors.NewObfuscateUpstreamErrors(cfg.ObfuscateUpstreamErrors)

	pxy, err := proxy.NewProxy(cfg.Target, blockFieldSuggestions, obfuscateUpstreamErrors, cfg.LogGraphqlErrors, log)
	if err != nil {
		log.Error("ErrorPayload creating proxy", "err", err)
		return nil
	}

	loader, err := trusteddocuments.NewLoaderFromConfig(cfg.PersistedOperations, log)
	if err != nil {
		log.Error("Error initializing persisted operations loader", "err", err)
		return err
	}

	po, err := trusteddocuments.NewPersistedOperations(log, cfg.PersistedOperations, loader)
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
	mux.Handle("/internal/debug_trusted_documents", debug.NewTrustedDocumentsDebugger(po, cfg.PersistedOperations.EnableDebugEndpoint))
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

func protectMiddlewareChain(log *slog.Logger) func(next http.Handler) http.Handler {
	rec := middleware.Recover(log)
	httpInstrumentation := middleware.RequestMetricMiddleware()
	otelHandler := otelhttp.NewMiddleware("GraphQL Protect")

	fn := func(next http.Handler) http.Handler {
		return rec(otelHandler(httpInstrumentation(next)))
	}

	return fn
}
