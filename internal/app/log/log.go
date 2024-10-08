package log

import (
	"log/slog"
	"os"
)

type Config struct {
	Format string `yaml:"format"`
}

func DefaultConfig() Config {
	return Config{
		Format: "json",
	}
}

var (
	JSONFormat = "json"
	TextFormat = "text"
)

func NewLogger(cfg Config) *slog.Logger {
	if cfg.Format == TextFormat {
		return slog.Default()
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}
