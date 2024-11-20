package trusteddocuments // nolint:revive

import (
	"context"
)

var _ Loader = &MemoryLoader{}

// MemoryLoader is a loader for testing purposes
// It allows the user to specify operations in memory
type MemoryLoader struct {
	Store map[string]PersistedOperation
}

func (d *MemoryLoader) Type() string {
	return "memory"
}

func newMemoryLoader(store map[string]PersistedOperation) *MemoryLoader {
	return &MemoryLoader{
		Store: store,
	}
}

func (d *MemoryLoader) Load(_ context.Context) (map[string]PersistedOperation, error) {
	return d.Store, nil
}
