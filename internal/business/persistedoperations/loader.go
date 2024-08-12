package persistedoperations // nolint:revive

import (
	"context"
	"log/slog"
)

type Loader interface {
	Load(ctx context.Context) (map[string]PersistedOperation, error)
	Type() string
}

func NewLoaderFromConfig(cfg Config, log *slog.Logger) (Loader, error) {
	switch cfg.Loader.Type {
	case "local":
		return NewLocalDirLoader(cfg, log), nil
	case "gcp":
		return NewGcpLoader(cfg.Loader, log)
	default:
		log.Info("Loader strategy defaulted to noop loader for type", "type", cfg.Loader.Type)
		return NewNoOpLoader()
	}
}
