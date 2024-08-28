package main

import (
	"flag"
	"github.com/ardanlabs/conf/v3"
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

	// cfg
	cfg, err := config.NewConfig(configPath)
	if err != nil {
		log2.Println("Error loading application configuration", "err", err)
		os.Exit(1)
	}
	cfgAsString, _ := conf.String(cfg)
	log2.Println(cfgAsString)

	logger := log.NewLogger(cfg.Log)
	logger.Info("Starting Protect", "version", build)

	appInfo.With(prometheus.Labels{
		"version":    build,
		"go_version": runtime.Version(),
		"short_hash": shortHash,
	})

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	action := os.Args[1]

	switch action {
	case "serve":
		if err := httpServer(logger, cfg, shutdown); err != nil {
			logger.Error("startup", "msg", err)
			os.Exit(1)
		}
	case "validate":
		if err := validate(logger, cfg, shutdown); err != nil {
			logger.Error("validate", "msg", err)
			os.Exit(1)
		}
	case "version":
		logger.Info("GraphQL Protect", "version", build, "go_version", runtime.Version(), "short_hash", shortHash)
		os.Exit(0)
	default:
		logger.Error("Subcommand required", "subcommand", action)
		os.Exit(1)
	}
}
