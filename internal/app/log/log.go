package log

import (
	"log/slog"
	"os"
)

type Config struct {
	Format string `conf:"default:json"`
}

var (
	JSONFormat string = "json"
	TextFormat string = "text"
)

func NewLogger(cfg Config) *slog.Logger {
	if cfg.Format == TextFormat {
		return slog.Default()
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}
