package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/ldebruijn/graphql-protect/internal/app/config"
	"github.com/ldebruijn/graphql-protect/internal/app/log"
	"github.com/prometheus/client_golang/prometheus"
	log2 "log"
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

	if len(os.Args) < 2 {
		log2.Println("Subcommand expected. Options are `serve`, `validate`, `version` or `help`")
		os.Exit(1)
	}
	action := os.Args[1]

	err := startup(action, configPath)
	if err != nil {
		log2.Println("Subcommand expected. Options are `serve`, `validate`, `version` or `help`")
		os.Exit(1)
	}
	os.Exit(0)
}

func startup(action string, path string) error {
	// cfg
	cfg, err := config.NewConfig(path)
	if err != nil {
		log2.Println("Error loading application configuration", "err", err)
		if !errors.Is(err, config.ErrConfigFileNotFound) {
			return err
		}
	}
	log2.Println("Configuration:")
	log2.Println(cfg)

	logger := log.NewLogger(cfg.Log)
	logger.Info("Starting Protect", "version", build)

	appInfo.With(prometheus.Labels{
		"version":    build,
		"go_version": runtime.Version(),
		"short_hash": shortHash,
	})

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	switch action {
	case "serve":
		if err := httpServer(logger, cfg, shutdown); err != nil {
			logger.Error("serve", "msg", err)
			return err
		}
	case "validate":
		if err := validate(logger, cfg, shutdown); err != nil {
			logger.Error("validate", "msg", err)
			return err
		}
	case "version":
		logger.Info("GraphQL Protect", "version", build, "go_version", runtime.Version(), "short_hash", shortHash)
	default:
		out := fmt.Sprintf("unexpeced subcommand, options are `serve`, `validate`, `version`. got: `%s`", action)
		logger.Error(out)
		return errors.New(out)
	}
	return nil
}
