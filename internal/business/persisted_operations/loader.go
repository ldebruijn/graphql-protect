package persisted_operations

import (
	"context"
	"errors"
)

var ErrNoLoaderSpecified = errors.New("no loaders specified")

// DetermineLoaderFromConfig looks at the configuration applied and figures out which loader to initialize and return
// If no loader is configured an error is returned
func DetermineLoaderFromConfig(cfg Config) (PersistedOperationsLoader, error) {
	loader, err := determineLoader(cfg)
	if err != nil {
		return loader, err
	}
	return loader, nil
}

// load loads persisted operations from various sources
func determineLoader(cfg Config) (PersistedOperationsLoader, error) {
	if cfg.Store.Dir != "" {
		loader := newDirLoader(cfg)
		if loader == nil {
			return nil, errors.New("unable to instantiate DirLoader")
		}

		return loader, nil
	}
	if cfg.Store.GcpBucket != "" {
		loader, err := NewGcpStorageLoader(context.Background(), cfg.Store.GcpBucket)
		if err != nil {
			return nil, errors.New("unable to instantiate GcpBucketLoader")
		}
		return loader, nil
	}
	return newMemoryLoader(map[string]string{}), ErrNoLoaderSpecified
}
