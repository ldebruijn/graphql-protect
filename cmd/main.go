package main

import (
	"context"
	"fmt"
	"github.com/ldebruijn/go-graphql-armor/internal/app/config"
	"github.com/ldebruijn/go-graphql-armor/internal/business/persisted_operations"
	"github.com/ldebruijn/go-graphql-armor/internal/business/proxy"
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
	ctx := context.Background()

	log := slog.Default()

	if err := run(ctx, log); err != nil {
		log.Error("startup", "msg", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *slog.Logger) error {
	log.Info("startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))

	// cfg
	cfg, err := config.NewConfig()
	if err != nil {
		log.Error("Error loading application configuration", "err", err)
		return nil
	}
	log.Info("Starting service", "version", build)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	log.Info("Starting proxy", "target", cfg.Target.Host)

	pxy, err := proxy.NewProxy(cfg.Target)
	if err != nil {
		log.Error("Error creating proxy", "err", err)
		return nil
	}

	mux := http.NewServeMux()

	mid := middleware(log, cfg)
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

		if err := api.Shutdown(ctx); err != nil {
			_ = api.Close()
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}

	return nil
}

func middleware(log *slog.Logger, cfg *config.Config) func(next http.Handler) http.Handler {
	poLoader, err := persisted_operations.DetermineLoaderFromConfig(cfg.PersistedOperations)
	if err != nil {
		// handle better
		log.Error("Unable to determine loading strategy for persisted operations", "err", err)
	}
	po, _ := persisted_operations.NewPersistedOperations(log, cfg.PersistedOperations, poLoader)

	fn := func(next http.Handler) http.Handler {
		return po.Execute(next)
	}

	return fn
}

func Handler(p *httputil.ReverseProxy) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		p.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
