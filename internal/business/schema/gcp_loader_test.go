package schema

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockObjectReader struct {
	content string
	err     error
}

func (m *mockObjectReader) read(_ context.Context) (string, error) {
	return m.content, m.err
}

func newMockGcpSchemaLoader(content string, err error) *gcpSchemaLoader {
	return &gcpSchemaLoader{
		reader: &mockObjectReader{content: content, err: err},
		bucket: "test-bucket",
		object: "schema.graphql",
		log:    slog.Default(),
	}
}

func TestGcpSchemaLoader_Fetch(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		readerErr   error
		wantContent string
		wantErr     bool
	}{
		{
			name:        "returns schema content on success",
			content:     minimalSchema,
			wantContent: minimalSchema,
		},
		{
			name:      "propagates reader error",
			readerErr: errors.New("gcs read failed"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := newMockGcpSchemaLoader(tt.content, tt.readerErr)

			got, err := loader.fetch()

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantContent, got)
		})
	}
}

func TestProvider_WithGcpLoader(t *testing.T) {
	// Verifies the wiring between gcpSchemaLoader and Provider without real GCS credentials.
	// This mirrors exactly what NewSchema does when loader.type == "gcp".
	p := &Provider{
		done: make(chan bool, 1),
		log:  slog.Default(),
	}

	gcpLoader := newMockGcpSchemaLoader(minimalSchema, nil)
	p.loadFn = func() error {
		contents, err := gcpLoader.fetch()
		if err != nil {
			return err
		}
		return p.load(contents)
	}

	require.NoError(t, p.loadFn())
	assert.NotNil(t, p.Get())
}

func TestProvider_WithGcpLoader_FetchError(t *testing.T) {
	p := &Provider{
		done: make(chan bool, 1),
		log:  slog.Default(),
	}

	gcpLoader := newMockGcpSchemaLoader("", errors.New("bucket not found"))
	p.loadFn = func() error {
		contents, err := gcpLoader.fetch()
		if err != nil {
			return err
		}
		return p.load(contents)
	}

	assert.Error(t, p.loadFn())
	assert.Nil(t, p.Get())
}
