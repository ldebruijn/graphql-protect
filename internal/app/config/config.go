package config

import (
	"errors"
	"fmt"
	"github.com/ardanlabs/conf/v3"
	"github.com/ardanlabs/conf/v3/yaml"
	"github.com/ldebruijn/go-graphql-armor/internal/business/aliases"
	"github.com/ldebruijn/go-graphql-armor/internal/business/block_field_suggestions"
	"github.com/ldebruijn/go-graphql-armor/internal/business/persisted_operations"
	"github.com/ldebruijn/go-graphql-armor/internal/business/proxy"
	"log/slog"
	"os"
	"time"
)

type Config struct {
	Web struct {
		ReadTimeout     time.Duration `conf:"default:5s" yaml:"read_timeout"`
		WriteTimeout    time.Duration `conf:"default:10s" yaml:"write_timeout"`
		IdleTimeout     time.Duration `conf:"default:120s" yaml:"idle_timeout"`
		ShutdownTimeout time.Duration `conf:"default:20s" yaml:"shutdown_timeout"`
		Host            string        `conf:"default:0.0.0.0:8080" yaml:"host"`
		// or maybe we just want to listen on everything and forward
		Path string `conf:"default:/graphql" yaml:"path"`
		//DebugHost       string        `conf:"default:0.0.0.0:4000"`
	}
	Target                proxy.Config                   `yaml:"target"`
	PersistedOperations   persisted_operations.Config    `yaml:"persisted_operations"`
	BlockFieldSuggestions block_field_suggestions.Config `yaml:"block_field_suggestions"`
	MaxAliases            aliases.Config                 `yaml:"max_aliases"`
}

func NewConfig(log *slog.Logger, configPath string) (*Config, error) {
	cfg := Config{}

	// ignore yaml read failure
	fromYaml, err := os.ReadFile(configPath)
	if err != nil && configPath != "" {
		log.Warn("Error loading configuration from filepath", configPath, err)
	}

	help, err := conf.Parse("go-graphql-armor", &cfg, yaml.WithData(fromYaml))
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
			return nil, conf.ErrHelpWanted
		}
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}
