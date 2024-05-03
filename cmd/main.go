package main

import (
	"flag"
	"github.com/ardanlabs/conf/v3"
	"github.com/ldebruijn/graphql-protect/internal/app/config"
	"github.com/prometheus/client_golang/prometheus"
	log2 "log"
	"log/slog"
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
		if action == "-f" {
			// backwards compatible change
			action = "serve"
		}
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
