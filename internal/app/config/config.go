package config

import (
	"errors"
	"fmt"
	"github.com/ardanlabs/conf/v3"
	"github.com/ldebruijn/go-graphql-armor/internal/business/persisted_operations"
	"github.com/ldebruijn/go-graphql-armor/internal/business/proxy"
	"time"
)

// Config struct for webapp config
type YConfig struct {
	Server struct {
		// Host is the local machine IP Address to bind the HTTP Server to
		Host string `yaml:"host"`

		// Port is the local machine TCP Port to bind the HTTP Server to
		Port    string `yaml:"port"`
		Timeout struct {
			// Server is the general server timeout to use
			// for graceful shutdowns
			Server time.Duration `yaml:"server"`

			// Write is the amount of time to wait until an HTTP server
			// write opperation is cancelled
			Write time.Duration `yaml:"write"`

			// Read is the amount of time to wait until an HTTP server
			// read operation is cancelled
			Read time.Duration `yaml:"read"`

			// Read is the amount of time to wait
			// until an IDLE HTTP session is closed
			Idle time.Duration `yaml:"idle"`
		} `yaml:"timeout"`
	} `yaml:"server"`
}

// rewrite to yaml
type Config struct {
	Web struct {
		ReadTimeout     time.Duration `conf:"default:5s"`
		WriteTimeout    time.Duration `conf:"default:10s"`
		IdleTimeout     time.Duration `conf:"default:120s"`
		ShutdownTimeout time.Duration `conf:"default:20s"`
		Host            string        `conf:"default:0.0.0.0:8080"`
		// or maybe we just want to listen on everything and forward
		Path string `conf:"default:/graphql"`
		//DebugHost       string        `conf:"default:0.0.0.0:4000"`
	}
	Target              proxy.Config
	PersistedOperations persisted_operations.Config
}

func NewConfig() (*Config, error) {
	cfg := Config{}

	help, err := conf.Parse("go-graphql-armor", &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
			return nil, conf.ErrHelpWanted
		}
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}
