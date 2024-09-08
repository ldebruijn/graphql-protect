package config

import (
	"errors"
	"github.com/ldebruijn/graphql-protect/internal/app/http"
	"github.com/ldebruijn/graphql-protect/internal/app/log"
	"github.com/ldebruijn/graphql-protect/internal/business/persistedoperations"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/accesslogging"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/aliases"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/batch"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/block_field_suggestions"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/enforce_post"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/max_depth"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/tokens"
	"github.com/ldebruijn/graphql-protect/internal/business/schema"
	"github.com/ldebruijn/graphql-protect/internal/http/proxy"
	y "gopkg.in/yaml.v3"
	"os"
	"time"
)

var ErrConfigFileNotFound = errors.New("config file could not be found, defaults applied")

type Config struct {
	Web                       http.Config                    `yaml:"web"`
	Schema                    schema.Config                  `yaml:"schema"`
	Target                    proxy.Config                   `yaml:"target"`
	PersistedOperations       persistedoperations.Config     `yaml:"persisted_operations"`
	ObfuscateValidationErrors bool                           `conf:"default:false" yaml:"obfuscate_validation_errors"`
	ObfuscateUpstreamErrors   bool                           `conf:"default:true" yaml:"obfuscate_upstream_errors"`
	BlockFieldSuggestions     block_field_suggestions.Config `yaml:"block_field_suggestions"`
	MaxTokens                 tokens.Config                  `yaml:"max_tokens"`
	MaxAliases                aliases.Config                 `yaml:"max_aliases"`
	EnforcePost               enforce_post.Config            `yaml:"enforce_post"`
	MaxDepth                  max_depth.Config               `yaml:"max_depth"`
	MaxBatch                  batch.Config                   `yaml:"max_batch"`
	AccessLogging             accesslogging.Config           `yaml:"access_logging"`
	Log                       log.Config                     `yaml:"log"`
	LogGraphqlErrors          bool                           `conf:"default:false" yaml:"log_graphql_errors"`
}

func NewConfig(configPath string) (*Config, error) {
	cfg := defaults()

	bts, err := os.ReadFile(configPath)
	if err != nil {
		return &cfg, errors.Join(ErrConfigFileNotFound, err)
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

func defaults() Config {
	return Config{
		Web:                       http.DefaultConfig(),
		Schema:                    schema.DefaultConfig(),
		Target:                    proxy.DefaultConfig(),
		PersistedOperations:       persistedoperations.DefaultConfig(),
		ObfuscateValidationErrors: false,
		ObfuscateUpstreamErrors:   true,
		BlockFieldSuggestions:     block_field_suggestions.DefaultConfig(),
		MaxTokens:                 tokens.DefaultConfig(),
		MaxAliases:                aliases.DefaultConfig(),
		EnforcePost:               enforce_post.DefaultConfig(),
		MaxDepth:                  max_depth.DefaultConfig(),
		MaxBatch:                  batch.DefaultConfig(),
		AccessLogging:             accesslogging.DefaultConfig(),
		Log:                       log.DefaultConfig(),
		LogGraphqlErrors:          false,
	}
}
