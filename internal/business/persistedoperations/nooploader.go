package persistedoperations

import "context"

var _ Loader = &NoOpLoader{}

type NoOpLoader struct {
}

func (n *NoOpLoader) Type() string {
	return "noop"
}

func NewNoOpLoader() (*NoOpLoader, error) {
	return &NoOpLoader{}, nil
}

func (n *NoOpLoader) Load(ctx context.Context) (map[string]PersistedOperation, error) {
	return nil, nil
}
