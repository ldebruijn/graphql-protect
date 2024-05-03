package persisted_operations // nolint:revive

import (
	"context"
)

// MemoryLoader is a loader for testing purposes
// It allows the user to specify operations in memory
type MemoryLoader struct {
	store map[string]PersistedOperation
}

func newMemoryLoader(store map[string]PersistedOperation) *MemoryLoader {
	return &MemoryLoader{
		store: store,
	}
}

func (d *MemoryLoader) Load(_ context.Context) (map[string]PersistedOperation, error) {
	return d.store, nil
}
