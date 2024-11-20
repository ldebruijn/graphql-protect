package config

import (
	"github.com/ldebruijn/graphql-protect/internal/app/http"
	"github.com/ldebruijn/graphql-protect/internal/app/log"
	"github.com/ldebruijn/graphql-protect/internal/business/persistedoperations"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/accesslogging"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/aliases"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/batch"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/block_field_suggestions"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/enforce_post"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/max_depth"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/tokens"
	"github.com/ldebruijn/graphql-protect/internal/business/schema"
	"github.com/ldebruijn/graphql-protect/internal/http/proxy"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"os"
	"testing"
	"time"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name        string
		applyConfig func(file *os.File)
		want        *Config
		wantErr     bool
	}{
		{
			name: "Assures defaults are applied correctly",
			applyConfig: func(_ *os.File) {

			},
			want: func() *Config {
				cfg := defaults()
				return &cfg
			}(),
			wantErr: false,
		},
		{
			name: "YAML overrides are applied",
			applyConfig: func(file *os.File) {
				_, _ = file.Write([]byte(`
web:
  read_timeout: 1s
  write_timeout: 1s
  idle_timeout: 1s
  shutdown_timeout: 1s
  host: host
  path: path
  request_body_max_bytes: 2048

target:
  host: host
  timeout: 1s
  keep_alive: 1s

schema:
  path: "path"
  auto_reload:
    enabled: false
    interval: 1s
    
obfuscate_validation_errors: true    
obfuscate_upstream_errors: false
    
persisted_operations:
  enabled: true
  enable_debug_endpoint: true
  reject_on_failure: false
  loader:
    type: gcp
    location: some-bucket
    reload:
      enabled: true
      interval: 1s
      timeout: 1s

max_aliases:
  enabled: false
  max: 1
  reject_on_failure: false

block_field_suggestions:
  enabled: false
  mask: mask
  
max_depth:
  enabled: false
  max: 1
  reject_on_failure: false
  field:
    enabled: false
    max: 1
    reject_on_failure: false
  list:
    enabled: false
    max: 1
    reject_on_failure: false

max_tokens:
  enabled: false
  max: 1
  reject_on_failure: false

max_batch:
  enabled: false
  max: 1
  reject_on_failure: false

enforce_post:
  enabled: false

access_logging:
  enabled: false
  include_headers:
    - Authorization
  include_operation_name: false
  include_variables: false
  include_payload: true

log:
  format: text

`))
			},
			want: &Config{
				Web: http.Config{
					ReadTimeout:         1 * time.Second,
					WriteTimeout:        1 * time.Second,
					IdleTimeout:         1 * time.Second,
					ShutdownTimeout:     1 * time.Second,
					Host:                "host",
					Path:                "path",
					RequestBodyMaxBytes: 2048,
				},
				ObfuscateValidationErrors: true,
				ObfuscateUpstreamErrors:   false,
				Schema: schema.Config{
					Path: "path",
					AutoReload: struct {
						Enabled  bool          `yaml:"enabled"`
						Interval time.Duration `yaml:"interval"`
					}(struct {
						Enabled  bool
						Interval time.Duration
					}{Enabled: false, Interval: 1 * time.Second}),
				},
				Target: proxy.Config{
					Timeout:   1 * time.Second,
					KeepAlive: 1 * time.Second,
					Host:      "host",
				},
				PersistedOperations: persistedoperations.Config{
					Enabled:             true,
					EnableDebugEndpoint: true,
					Loader: persistedoperations.LoaderConfig{
						Type:     "gcp",
						Location: "some-bucket",
						Reload: struct {
							Enabled  bool          `yaml:"enabled"`
							Interval time.Duration `yaml:"interval"`
							Timeout  time.Duration `yaml:"timeout"`
						}{
							Enabled:  true,
							Interval: 1 * time.Second,
							Timeout:  1 * time.Second,
						},
					},
					RejectOnFailure: false,
				},
				BlockFieldSuggestions: block_field_suggestions.Config{
					Enabled: false,
					Mask:    "mask",
				},
				MaxTokens: tokens.Config{
					Enabled:         false,
					Max:             1,
					RejectOnFailure: false,
				},
				MaxAliases: aliases.Config{
					Enabled:         false,
					Max:             1,
					RejectOnFailure: false,
				},
				EnforcePost: enforce_post.Config{
					Enabled: false,
				},
				MaxDepth: max_depth.Config{
					Enabled:         false,
					Max:             1,
					RejectOnFailure: false,
					Field: max_depth.MaxRule{
						Enabled:         false,
						Max:             1,
						RejectOnFailure: false,
					},
					List: max_depth.MaxRule{
						Enabled:         false,
						Max:             1,
						RejectOnFailure: false,
					},
				},
				MaxBatch: batch.Config{
					Enabled:         false,
					Max:             1,
					RejectOnFailure: false,
				},
				AccessLogging: accesslogging.Config{
					Enabled:              false,
					IncludedHeaders:      []string{"Authorization"},
					IncludeOperationName: false,
					IncludeVariables:     false,
					IncludePayload:       true,
				},
				Log: log.Config{
					Format: log.TextFormat,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, _ := os.CreateTemp("", "")
			defer func() {
				_ = os.Remove(file.Name())
			}()

			tt.applyConfig(file)

			got, err := NewConfig(file.Name())
			assert.NoError(t, err)

			assert.Equal(t, tt.want, got)
		})
	}
}

// WriteDefaultConfigToYaml is used to write a configuration file with pure defaults to a yaml file.
// This makes it really easy to copy-paste it onto documentation examples.
func TestWriteDefaultConfigToYaml(t *testing.T) {
	t.Skip("not actually a test, abusing the test for easy generation of default configuration file")

	cfg := defaults()

	file, err := os.OpenFile("default-config.yml", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		assert.NoError(t, err)
		return
	}
	defer file.Close()

	enc := yaml.NewEncoder(file)

	_ = enc.Encode(cfg)
}
