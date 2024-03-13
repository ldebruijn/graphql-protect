package persisted_operations // nolint:revive

import (
	"context"
	"errors"
	"log/slog"
)

type LocalLoader interface {
	Load(ctx context.Context) (map[string]string, error)
}

type RemoteLoader interface {
	Load(ctx context.Context) error
}

var ErrNoRemoteLoaderSpecified = errors.New("no remote loader specified")

// RemoteLoaderFromConfig looks at the configuration applied and figures out which remoteLoader to initialize and return
// If no remoteLoader is configured an error is returned
func RemoteLoaderFromConfig(cfg Config, log *slog.Logger) (RemoteLoader, error) {
	loader, err := determineLoader(cfg, log)
	if err != nil {
		return loader, err
	}
	return loader, nil
}

// load loads persisted operations from various sources
func determineLoader(cfg Config, log *slog.Logger) (RemoteLoader, error) {
	if cfg.Remote.GcpBucket != "" {
		loader, err := NewGcpStorageLoader(context.Background(), cfg.Remote.GcpBucket, cfg.Store, log)
		if err != nil {
			return nil, errors.New("unable to instantiate GcpBucketLoader")
		}
		return loader, nil
	}
	return nil, ErrNoRemoteLoaderSpecified
}
