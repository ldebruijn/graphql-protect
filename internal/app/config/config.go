package config

import (
	"errors"
	"fmt"
	"github.com/ardanlabs/conf/v3"
	"github.com/ardanlabs/conf/v3/yaml"
	"github.com/ldebruijn/graphql-protect/internal/business/persisted_operations"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/aliases"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/batch"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/block_field_suggestions"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/enforce_post"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/max_depth"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/tokens"
	"github.com/ldebruijn/graphql-protect/internal/business/schema"
	"github.com/ldebruijn/graphql-protect/internal/http/proxy"
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
		// DebugHost       string        `conf:"default:0.0.0.0:4000"`
	}
	ObfuscateValidationErrors bool                           `conf:"default:false" yaml:"obfuscate_validation_errors"`
	Schema                    schema.Config                  `yaml:"schema"`
	Target                    proxy.Config                   `yaml:"target"`
	PersistedOperations       persisted_operations.Config    `yaml:"persisted_operations"`
	BlockFieldSuggestions     block_field_suggestions.Config `yaml:"block_field_suggestions"`
	MaxTokens                 tokens.Config                  `yaml:"max_tokens"`
	MaxAliases                aliases.Config                 `yaml:"max_aliases"`
	EnforcePost               enforce_post.Config            `yaml:"enforce_post"`
	MaxDepth                  max_depth.Config               `yaml:"max_depth"`
	MaxBatch                  batch.Config                   `yaml:"max_batch"`
}

func NewConfig(configPath string) (*Config, error) {
	cfg := Config{}

	help, err := conf.Parse("graphql-protect", &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
			return nil, conf.ErrHelpWanted
		}
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if configPath != "" {
		// ignore yaml read failure
		fromYaml, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("could not read config file [%s], %w", configPath, err)
		}

		// process yaml after parse, set defaults first and override with yaml after
		err = yaml.WithData(fromYaml).Process("", &cfg)
		if err != nil {
			return nil, err
		}
	}

	return &cfg, nil
}
