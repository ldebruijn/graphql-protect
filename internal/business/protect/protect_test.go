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
	"strings"
	"testing"
	"time"
)

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
				accessLogging: accesslogging.NewAccessLogging(accesslogging.Config{}, log),
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
				accessLogging: accesslogging.NewAccessLogging(accesslogging.Config{}, log),
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
				accessLogging: accesslogging.NewAccessLogging(accesslogging.Config{}, log),
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
