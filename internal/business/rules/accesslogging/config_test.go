package accesslogging

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v3"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.True(t, cfg.Enabled)
	assert.True(t, cfg.IncludeOperationName)
	assert.True(t, cfg.IncludeVariables)
	assert.False(t, cfg.IncludePayload)
	assert.False(t, cfg.Async)
	assert.Equal(t, 1000, cfg.BufferSize)

	gcp := cfg.GoogleCloudLogging
	assert.False(t, gcp.Enabled)
	assert.Empty(t, gcp.ProjectID)
	assert.Empty(t, gcp.LogName)
	// Preserves the previous GCP client default (1000 MB ≈ 1GiB) so upgrades are behaviour-neutral.
	assert.Equal(t, 1000, gcp.BufferedByteLimit)
}

// TestGoogleCloudConfig_YAMLBinding guards the yaml tag names against typos:
// a wrong tag would silently leave fields at their zero value.
func TestGoogleCloudConfig_YAMLBinding(t *testing.T) {
	raw := []byte(`
enabled: true
project_id: "my-gcp-project"
log_name: "my-logs"
log_buffer_max_size_mb: 256
`)

	var cfg GoogleCloudConfig
	require.NoError(t, yaml.Unmarshal(raw, &cfg))

	assert.True(t, cfg.Enabled)
	assert.Equal(t, "my-gcp-project", cfg.ProjectID)
	assert.Equal(t, "my-logs", cfg.LogName)
	assert.Equal(t, 256, cfg.BufferedByteLimit)
}

// TestGoogleCloudConfig_YAMLOmittedKeepsDefault documents that unmarshalling over a
// pre-populated struct (as the loader does after DefaultConfig) preserves the default
// when log_buffer_max_size_mb is omitted.
func TestGoogleCloudConfig_YAMLOmittedKeepsDefault(t *testing.T) {
	cfg := DefaultConfig().GoogleCloudLogging

	raw := []byte(`
enabled: true
project_id: "my-gcp-project"
`)
	require.NoError(t, yaml.Unmarshal(raw, &cfg))

	assert.Equal(t, 1000, cfg.BufferedByteLimit)
}
