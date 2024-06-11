package config

import (
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
			want: &Config{
				Web: struct {
					ReadTimeout     time.Duration `conf:"default:5s" yaml:"read_timeout"`
					WriteTimeout    time.Duration `conf:"default:10s" yaml:"write_timeout"`
					IdleTimeout     time.Duration `conf:"default:120s" yaml:"idle_timeout"`
					ShutdownTimeout time.Duration `conf:"default:20s" yaml:"shutdown_timeout"`
					Host            string        `conf:"default:0.0.0.0:8080" yaml:"host"`
					Path            string        `conf:"default:/graphql" yaml:"path"`
				}(struct {
					ReadTimeout     time.Duration
					WriteTimeout    time.Duration
					IdleTimeout     time.Duration
					ShutdownTimeout time.Duration
					Host            string
					Path            string
				}{
					ReadTimeout:     5 * time.Second,
					WriteTimeout:    10 * time.Second,
					IdleTimeout:     2 * time.Minute,
					ShutdownTimeout: 20 * time.Second,
					Host:            "0.0.0.0:8080",
					Path:            "/graphql",
				}),
				ObfuscateValidationErrors: false,
				Schema: schema.Config{
					Path: "./schema.graphql",
					AutoReload: struct {
						Enabled  bool          `conf:"default:true" yaml:"enabled"`
						Interval time.Duration `conf:"default:30s" yaml:"interval"`
					}(struct {
						Enabled  bool
						Interval time.Duration
					}{Enabled: true, Interval: 30 * time.Second}),
				},
				Target: proxy.Config{
					Timeout:   10 * time.Second,
					KeepAlive: 3 * time.Minute,
					Host:      "http://localhost:8081",
				},
				PersistedOperations: persistedoperations.Config{
					Enabled: false,
					Store:   "./store",
					Reload: struct {
						Enabled  bool          `conf:"default:true" yaml:"enabled"`
						Interval time.Duration `conf:"default:5m" yaml:"interval"`
						Timeout  time.Duration `conf:"default:10s" yaml:"timeout"`
					}(struct {
						Enabled  bool
						Interval time.Duration
						Timeout  time.Duration
					}{
						Enabled:  true,
						Interval: 5 * time.Minute,
						Timeout:  10 * time.Second,
					}),
					Remote: struct {
						GcpBucket string `yaml:"gcp_bucket"`
					}(struct{ GcpBucket string }{GcpBucket: ""}),
					RejectOnFailure: true,
				},
				BlockFieldSuggestions: block_field_suggestions.Config{
					Enabled: true,
					Mask:    "[redacted]",
				},
				MaxTokens: tokens.Config{
					Enabled:         true,
					Max:             1000,
					RejectOnFailure: true,
				},
				MaxAliases: aliases.Config{
					Enabled:         true,
					Max:             15,
					RejectOnFailure: true,
				},
				EnforcePost: enforce_post.Config{
					Enabled: true,
				},
				MaxDepth: max_depth.Config{
					Enabled:         true,
					Max:             15,
					RejectOnFailure: true,
				},
				MaxBatch: batch.Config{
					Enabled:         true,
					Max:             5,
					RejectOnFailure: true,
				},
				AccessLogging: accesslogging.Config{
					Enabled:              true,
					IncludedHeaders:      nil,
					IncludeOperationName: true,
					IncludeVariables:     true,
					IncludePayload:       false,
				},
			},
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
    
persisted_operations:
  enabled: true
  reject_on_failure: false
  store: "store"
  reload:
    enabled: true
    interval: 1s
    timeout: 1s
  remote:
    gcp_bucket: "gcp_bucket"

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
`))
			},
			want: &Config{
				Web: struct {
					ReadTimeout     time.Duration `conf:"default:5s" yaml:"read_timeout"`
					WriteTimeout    time.Duration `conf:"default:10s" yaml:"write_timeout"`
					IdleTimeout     time.Duration `conf:"default:120s" yaml:"idle_timeout"`
					ShutdownTimeout time.Duration `conf:"default:20s" yaml:"shutdown_timeout"`
					Host            string        `conf:"default:0.0.0.0:8080" yaml:"host"`
					Path            string        `conf:"default:/graphql" yaml:"path"`
				}(struct {
					ReadTimeout     time.Duration
					WriteTimeout    time.Duration
					IdleTimeout     time.Duration
					ShutdownTimeout time.Duration
					Host            string
					Path            string
				}{
					ReadTimeout:     1 * time.Second,
					WriteTimeout:    1 * time.Second,
					IdleTimeout:     1 * time.Second,
					ShutdownTimeout: 1 * time.Second,
					Host:            "host",
					Path:            "path",
				}),
				ObfuscateValidationErrors: true,
				Schema: schema.Config{
					Path: "path",
					AutoReload: struct {
						Enabled  bool          `conf:"default:true" yaml:"enabled"`
						Interval time.Duration `conf:"default:30s" yaml:"interval"`
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
					Enabled: true,
					Store:   "store",
					Reload: struct {
						Enabled  bool          `conf:"default:true" yaml:"enabled"`
						Interval time.Duration `conf:"default:5m" yaml:"interval"`
						Timeout  time.Duration `conf:"default:10s" yaml:"timeout"`
					}(struct {
						Enabled  bool
						Interval time.Duration
						Timeout  time.Duration
					}{
						Enabled:  true,
						Interval: 1 * time.Second,
						Timeout:  1 * time.Second,
					}),
					Remote: struct {
						GcpBucket string `yaml:"gcp_bucket"`
					}(struct{ GcpBucket string }{GcpBucket: "gcp_bucket"}),
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

	cfg, err := NewConfig("")
	if err != nil {
		assert.NoError(t, err)
		return
	}

	file, err := os.OpenFile("default-config.yml", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		assert.NoError(t, err)
		return
	}
	defer file.Close()

	enc := yaml.NewEncoder(file)

	_ = enc.Encode(cfg)
}
