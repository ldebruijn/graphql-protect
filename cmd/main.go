package main

import (
	"errors"
	"flag"
	"fmt"
	log2 "log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/ldebruijn/graphql-protect/internal/app/config"
	"github.com/ldebruijn/graphql-protect/internal/app/log"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	shortHash = "develop"
	build     = "develop"

	appInfo = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "graphql_protect",
		Subsystem: "app",
		Name:      "info",
		Help:      "Application information",
	},
		[]string{"version", "go_version", "short_hash"},
	)

	ErrNoSubCommand = errors.New("subcommand expected. options are `serve`, `validate`, `version` or `help`")
)

func init() {
	prometheus.MustRegister(appInfo)
}

func main() {
	action, configPath, err := parseFlags(os.Args)
	if err != nil {
		log2.Println(err)
		os.Exit(1)
		return
	}

	log2.Println("Reading configuration from", configPath)

	err = startup(action, configPath)
	if err != nil {
		log2.Println("Subcommand expected. Options are `serve`, `validate`, `version` or `help`")
		os.Exit(1)
	}
	os.Exit(0)
}

func parseFlags(args []string) (string, string, error) {
	if len(args) < 2 {
		return "", "", ErrNoSubCommand
	}
	log2.Println("Initialized with arguments: ", args)

	action := strings.ToLower(args[1])

	flagSet := flag.NewFlagSet("", flag.ContinueOnError)
	configPath := flagSet.String("f", "./protect.yml", "Defines the path at which the configuration file can be found")
	err := flagSet.Parse(args[2:])
	if err != nil {
		return "", "", err
	}
	return action, *configPath, nil
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
