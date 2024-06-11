package log

import (
	"github.com/ldebruijn/graphql-protect/internal/app/env"
	"log/slog"
	"os"
)

func NewLogger(environment env.Environment) *slog.Logger {
	if environment == env.Dev {
		return slog.Default()
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}
