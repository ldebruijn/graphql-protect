package http

import "time"

type Config struct {
	ReadTimeout     time.Duration `conf:"default:5s" yaml:"read_timeout"`
	WriteTimeout    time.Duration `conf:"default:10s" yaml:"write_timeout"`
	IdleTimeout     time.Duration `conf:"default:120s" yaml:"idle_timeout"`
	ShutdownTimeout time.Duration `conf:"default:20s" yaml:"shutdown_timeout"`
	Host            string        `conf:"default:0.0.0.0:8080" yaml:"host"`
	// or maybe we just want to listen on everything and forward
	Path string `conf:"default:/graphql" yaml:"path"`
	// DebugHost       string        `conf:"default:0.0.0.0:4000"`
}

func DefaultConfig() Config {
	return Config{
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    10 * time.Second,
		IdleTimeout:     2 * time.Minute,
		ShutdownTimeout: 20 * time.Second,
		Host:            "0.0.0.0:8080",
		Path:            "/graphql",
	}
}
