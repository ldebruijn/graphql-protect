package persistedoperations // nolint:revive

import (
	"context"
	"errors"
	"log/slog"
)

type Loader interface {
	Load(ctx context.Context) (map[string]PersistedOperation, error)
	Type() string
}

func LoaderFromConfig(cfg Config, log *slog.Logger) (Loader, error) {
	switch cfg.Loader.Type {
	case "local":
		return NewLocalDirLoader(cfg, log), nil
	case "gcp":
		return NewGcpLoader(cfg.Loader, log)
	default:
		return NewNoOpLoader()
	}
}

var ErrNoRemoteLoaderSpecified = errors.New("no remote loader specified")
