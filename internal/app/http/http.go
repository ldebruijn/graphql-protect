package http

import "time"

const kilobyte100 = 102_400 // 100kb

type Config struct {
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	Host            string        `yaml:"host"`
	// or maybe we just want to listen on everything and forward
	Path string `yaml:"path"`
	// DebugHost       string        `yaml:"debug_host"`
	RequestBodyMaxBytes int `yaml:"request_body_max_bytes"`
}

func DefaultConfig() Config {
	return Config{
		ReadTimeout:         5 * time.Second,
		WriteTimeout:        10 * time.Second,
		IdleTimeout:         2 * time.Minute,
		ShutdownTimeout:     20 * time.Second,
		Host:                "0.0.0.0:8080",
		Path:                "/graphql",
		RequestBodyMaxBytes: kilobyte100,
	}
}
