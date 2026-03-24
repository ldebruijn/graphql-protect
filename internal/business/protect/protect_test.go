package protect

import (
	"github.com/ldebruijn/graphql-protect/internal/app/config"
	_http "github.com/ldebruijn/graphql-protect/internal/app/http"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/accesslogging"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/batch"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/tokens"
	"github.com/ldebruijn/graphql-protect/internal/business/schema"
	"github.com/stretchr/testify/assert"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func mustNewAccessLogging(cfg accesslogging.Config, log *slog.Logger) *accesslogging.AccessLogging {
	al, err := accesslogging.NewAccessLogging(cfg, log)
	if err != nil {
		panic(err)
	}
	return al
}

func TestGraphQLProtect_ServeHTTP(t *testing.T) {
	log := slog.Default()

	type fields struct {
		log            *slog.Logger
		cfg            *config.Config
		schema         *schema.Provider
		tokens         *tokens.MaxTokensRule
		maxBatch       *batch.MaxBatchRule
		accessLogging  *accesslogging.AccessLogging
		next           http.Handler
		preFilterChain func(handler http.Handler) http.Handler
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "request body limit is respected",
			fields: fields{
				log: log,
				cfg: &config.Config{
					Web: _http.Config{
						ReadTimeout:         10 * time.Second,
						WriteTimeout:        10 * time.Second,
						IdleTimeout:         10 * time.Second,
						ShutdownTimeout:     10 * time.Second,
						Host:                "localhost",
						Path:                "/graphql",
						RequestBodyMaxBytes: 10,
					},
				},
				schema:        nil,
				tokens:        nil,
				maxBatch:      nil,
				accessLogging: mustNewAccessLogging(accesslogging.Config{}, log),
				next:          &noop{},
				preFilterChain: func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						next.ServeHTTP(w, r)
					})
				},
			},
			want: `{"data":null,"errors":[{"message":"http: request body too large"}]}
`,
		},
		{
			name: "limit of 0 means no limit",
			fields: fields{
				log: log,
				cfg: &config.Config{
					Web: _http.Config{
						ReadTimeout:         10 * time.Second,
						WriteTimeout:        10 * time.Second,
						IdleTimeout:         10 * time.Second,
						ShutdownTimeout:     10 * time.Second,
						Host:                "localhost",
						Path:                "/graphql",
						RequestBodyMaxBytes: 0,
					},
				},
				schema:        nil,
				tokens:        nil,
				maxBatch:      nil,
				accessLogging: mustNewAccessLogging(accesslogging.Config{}, log),
				next:          &noop{},
				preFilterChain: func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						next.ServeHTTP(w, r)
					})
				},
			},
			// this assertion doesn't test the actual intended result, but shows that the request body limitation does not affect this request
			// ideally this should be improved at a later stage.
			want: `{"data":null,"errors":[{"message":"invalid character 'i' looking for beginning of value"}]}
`,
		},
		{
			name: "request body limit does not interfere with request bodies with fewer bytes than the limit",
			fields: fields{
				log: log,
				cfg: &config.Config{
					Web: _http.Config{
						ReadTimeout:         10 * time.Second,
						WriteTimeout:        10 * time.Second,
						IdleTimeout:         10 * time.Second,
						ShutdownTimeout:     10 * time.Second,
						Host:                "localhost",
						Path:                "/graphql",
						RequestBodyMaxBytes: 1_000_000,
					},
				},
				schema:        nil,
				tokens:        nil,
				maxBatch:      nil,
				accessLogging: mustNewAccessLogging(accesslogging.Config{}, log),
				next:          &noop{},
				preFilterChain: func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						next.ServeHTTP(w, r)
					})
				},
			},
			// this assertion doesn't test the actual intended result, but shows that the request body limitation does not affect this request
			// ideally this should be improved at a later stage.
			want: `{"data":null,"errors":[{"message":"invalid character 'i' looking for beginning of value"}]}
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/graphql", strings.NewReader("i am a body that exceeds the reader limit"))

			p := &GraphQLProtect{
				log:            tt.fields.log,
				cfg:            tt.fields.cfg,
				schema:         tt.fields.schema,
				tokens:         tt.fields.tokens,
				maxBatch:       tt.fields.maxBatch,
				accessLogging:  tt.fields.accessLogging,
				next:           tt.fields.next,
				preFilterChain: tt.fields.preFilterChain,
			}
			p.ServeHTTP(w, r)

			res := w.Result()
			assert.Equal(t, res.StatusCode, http.StatusOK)

			body, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			defer res.Body.Close()

			assert.Equal(t, tt.want, string(body))
		})
	}
}

type noop struct {
}

func (n *noop) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}

func createTestSchemaProvider(t *testing.T) *schema.Provider {
	t.Helper()

	// Create temporary schema file
	tmpFile, err := os.CreateTemp("", "schema-*.graphql")
	if err != nil {
		t.Fatalf("Failed to create temp schema file: %v", err)
	}
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	schemaContent := `type Query { hello: String }`
	if _, err := tmpFile.WriteString(schemaContent); err != nil {
		t.Fatalf("Failed to write schema: %v", err)
	}
	tmpFile.Close()

	provider, err := schema.NewSchema(schema.Config{
		Path: tmpFile.Name(),
		AutoReload: struct {
			Enabled  bool          `yaml:"enabled"`
			Interval time.Duration `yaml:"interval"`
		}{
			Enabled:  false,
			Interval: 0,
		},
	}, slog.Default())
	if err != nil {
		t.Fatalf("Failed to create schema provider: %v", err)
	}

	return provider
}

func TestGraphQLProtect_TimingContextPropagation(t *testing.T) {
	log := slog.Default()

	// Create a handler that checks for TimingContext
	var capturedTC *TimingContext
	upstreamHandler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedTC = TimingContextFromContext(r.Context())
	})

	maxBatch, _ := batch.NewMaxBatch(batch.Config{
		Enabled:         true,
		Max:             10,
		RejectOnFailure: true,
	})

	schemaProvider := createTestSchemaProvider(t)

	p := &GraphQLProtect{
		log: log,
		cfg: &config.Config{
			Web: _http.Config{
				RequestBodyMaxBytes: 0,
			},
		},
		schema:        schemaProvider,
		maxBatch:      maxBatch,
		tokens:        tokens.MaxTokens(tokens.DefaultConfig()),
		accessLogging: accesslogging.NewAccessLogging(accesslogging.Config{}, log),
		next:          upstreamHandler,
		preFilterChain: func(next http.Handler) http.Handler {
			return next
		},
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/graphql", strings.NewReader(`{"query":"{ hello }"}`))

	p.ServeHTTP(w, r)

	assert.NotNil(t, capturedTC, "TimingContext should be propagated to upstream handler")
	assert.NotZero(t, capturedTC.Start, "Start time should be set")
}

func TestGraphQLProtect_MarkProtectEndBeforeUpstream(t *testing.T) {
	log := slog.Default()

	// Create a handler that checks End is marked
	var capturedTC *TimingContext
	upstreamHandler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedTC = TimingContextFromContext(r.Context())
	})

	maxBatch, _ := batch.NewMaxBatch(batch.Config{
		Enabled:         true,
		Max:             10,
		RejectOnFailure: true,
	})

	schemaProvider := createTestSchemaProvider(t)

	p := &GraphQLProtect{
		log: log,
		cfg: &config.Config{
			Web: _http.Config{
				RequestBodyMaxBytes: 0,
			},
		},
		schema:        schemaProvider,
		maxBatch:      maxBatch,
		tokens:        tokens.MaxTokens(tokens.DefaultConfig()),
		accessLogging: accesslogging.NewAccessLogging(accesslogging.Config{}, log),
		next:          upstreamHandler,
		preFilterChain: func(next http.Handler) http.Handler {
			return next
		},
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/graphql", strings.NewReader(`{"query":"{ hello }"}`))

	p.ServeHTTP(w, r)

	assert.NotNil(t, capturedTC, "TimingContext should exist")
	assert.False(t, capturedTC.End.IsZero(), "End should be marked before proxying to upstream")
	assert.True(t, capturedTC.End.After(capturedTC.Start), "End should be after Start")
}

func TestGraphQLProtect_DurationCalculation(t *testing.T) {
	log := slog.Default()

	// Capture timing context from upstream handler
	var capturedTC *TimingContext
	var upstreamStart time.Time

	upstreamHandler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedTC = TimingContextFromContext(r.Context())
		upstreamStart = time.Now()
		time.Sleep(50 * time.Millisecond)
	})

	maxBatch, _ := batch.NewMaxBatch(batch.Config{
		Enabled:         true,
		Max:             10,
		RejectOnFailure: true,
	})

	schemaProvider := createTestSchemaProvider(t)

	p := &GraphQLProtect{
		log: log,
		cfg: &config.Config{
			Web: _http.Config{
				RequestBodyMaxBytes: 0,
			},
		},
		schema:        schemaProvider,
		maxBatch:      maxBatch,
		tokens:        tokens.MaxTokens(tokens.DefaultConfig()),
		accessLogging: accesslogging.NewAccessLogging(accesslogging.Config{}, log),
		next:          upstreamHandler,
		preFilterChain: func(next http.Handler) http.Handler {
			return next
		},
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/graphql", strings.NewReader(`{"query":"{ hello }"}`))

	start := time.Now()
	p.ServeHTTP(w, r)
	totalDuration := time.Since(start)

	assert.NotNil(t, capturedTC, "TimingContext should exist")

	protectDuration := capturedTC.Duration()
	assert.Greater(t, totalDuration, protectDuration, "Total duration should be greater than protect duration")

	// Verify protect ended before upstream started
	assert.True(t, capturedTC.End.Before(upstreamStart) || capturedTC.End.Equal(upstreamStart),
		"End should be before or equal to upstream start")

	// Verify protect duration is positive
	assert.Greater(t, protectDuration, time.Duration(0), "Protect duration should be positive")
}
