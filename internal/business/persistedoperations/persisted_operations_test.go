package persistedoperations // nolint:revive

import (
	"bytes"
	"encoding/json"
	"github.com/ldebruijn/graphql-protect/internal/business/gql"
	"github.com/stretchr/testify/assert"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewPersistedOperations(t *testing.T) {
	type args struct {
		cfg     Config
		payload []byte
		cache   map[string]PersistedOperation
	}
	tests := []struct {
		name    string
		args    args
		want    func(t *testing.T) http.Handler
		resWant func(t *testing.T, res *http.Response)
	}{
		{
			name: "does nothing if middleware is disabled",
			args: args{
				cfg: Config{
					Enabled: false,
				},
			},
			want: func(_ *testing.T) http.Handler {
				fn := func(_ http.ResponseWriter, _ *http.Request) {
				}
				return http.HandlerFunc(fn)
			},
			resWant: func(_ *testing.T, _ *http.Response) {

			},
		},
		{
			name: "Allows unpersisted requests if configured",
			args: args{
				cfg: Config{
					Enabled:         true,
					RejectOnFailure: true,
				},
				payload: func() []byte {
					data := gql.RequestData{
						Query: "query { foo }",
					}
					bts, _ := json.Marshal(data)
					return bts
				}(),
			},
			want: func(t *testing.T) http.Handler {
				fn := func(_ http.ResponseWriter, r *http.Request) {
					decoder := json.NewDecoder(r.Body)

					var payload gql.RequestData
					err := decoder.Decode(&payload)
					assert.NoError(t, err)

					assert.Equal(t, "query { foo }", payload.Query)
				}
				return http.HandlerFunc(fn)
			},
			resWant: func(t *testing.T, res *http.Response) {
				assert.Equal(t, 200, res.StatusCode)
			},
		},
		{
			name: "Returns error if no hash match is found and unpersisted operations are not allowed",
			args: args{
				cfg: Config{
					Enabled:         true,
					RejectOnFailure: false,
				},
				payload: func() []byte {
					data := gql.RequestData{
						Extensions: gql.Extensions{
							PersistedQuery: &gql.PersistedQuery{
								Sha256Hash: "foobar",
							},
						},
					}
					bts, _ := json.Marshal(data)
					return bts
				}(),

				cache: map[string]PersistedOperation{},
			},
			want: func(_ *testing.T) http.Handler {
				fn := func(_ http.ResponseWriter, _ *http.Request) {

				}
				return http.HandlerFunc(fn)
			},
			resWant: func(t *testing.T, res *http.Response) {
				assert.Equal(t, 200, res.StatusCode)

				decoder := json.NewDecoder(res.Body)

				var payload ErrorPayload
				err := decoder.Decode(&payload)
				assert.NoError(t, err)

				assert.Equal(t, "PersistedOperationNotFound", payload.Errors[0].Message)
			},
		},
		{
			name: "Swaps in query payload if hash operation is known, updates content length accordingly",
			args: args{
				cfg: Config{
					Enabled:         true,
					RejectOnFailure: false,
				},
				payload: func() []byte {
					data := gql.RequestData{
						Extensions: gql.Extensions{
							PersistedQuery: &gql.PersistedQuery{
								Sha256Hash: "foobar",
							},
						},
					}
					bts, _ := json.Marshal(data)
					return bts
				}(),
				cache: map[string]PersistedOperation{
					"foobar": NewPersistedOperation("query { foobar }"),
				},
			},
			want: func(t *testing.T) http.Handler {
				fn := func(_ http.ResponseWriter, r *http.Request) {
					decoder := json.NewDecoder(r.Body)

					var payload gql.RequestData
					err := decoder.Decode(&payload)
					assert.NoError(t, err)

					assert.Equal(t, "query { foobar }", payload.Query)
					assert.Equal(t, int64(44), r.ContentLength)

					length, _ := json.Marshal(payload)

					assert.Equal(t, 44, len(length))
				}
				return http.HandlerFunc(fn)
			},
			resWant: func(t *testing.T, res *http.Response) {
				assert.Equal(t, 200, res.StatusCode)
			},
		},
		{
			name: "Swaps in batched query payload if hash operation is known, updates content length accordingly",
			args: args{
				cfg: Config{
					Enabled:         true,
					RejectOnFailure: false,
				},
				payload: func() []byte {
					data := []gql.RequestData{
						{
							Extensions: gql.Extensions{
								PersistedQuery: &gql.PersistedQuery{
									Sha256Hash: "foobar",
								},
							},
						},
						{
							Extensions: gql.Extensions{
								PersistedQuery: &gql.PersistedQuery{
									Sha256Hash: "baz",
								},
							},
						},
					}
					bts, _ := json.Marshal(data)
					return bts
				}(),
				cache: map[string]PersistedOperation{
					"foobar": NewPersistedOperation("query { foobar }"),
					"baz":    NewPersistedOperation("query { baz }"),
				},
			},
			want: func(t *testing.T) http.Handler {
				fn := func(_ http.ResponseWriter, r *http.Request) {
					payload, err := io.ReadAll(r.Body)
					assert.NoError(t, err)

					assert.Equal(t, `[{"query":"query { foobar }","extensions":{}},{"query":"query { baz }","extensions":{}}]`, string(payload))
					assert.Equal(t, int64(len(payload)), r.ContentLength)
				}
				return http.HandlerFunc(fn)
			},
			resWant: func(t *testing.T, res *http.Response) {
				assert.Equal(t, 200, res.StatusCode)
			},
		},
		{
			name: "fails entire batch if one operation is unknown",
			args: args{
				cfg: Config{
					Enabled:         true,
					RejectOnFailure: false,
				},
				payload: func() []byte {
					data := []gql.RequestData{
						{
							Extensions: gql.Extensions{
								PersistedQuery: &gql.PersistedQuery{
									Sha256Hash: "foobar",
								},
							},
						},
						{
							Extensions: gql.Extensions{
								PersistedQuery: &gql.PersistedQuery{
									Sha256Hash: "baz",
								},
							},
						},
					}
					bts, _ := json.Marshal(data)
					return bts
				}(),
				cache: map[string]PersistedOperation{
					"foobar": NewPersistedOperation("query { foobar }"),
				},
			},
			want: func(t *testing.T) http.Handler {
				fn := func(_ http.ResponseWriter, _ *http.Request) {
					assert.Fail(t, "should not reach here")
				}
				return http.HandlerFunc(fn)
			},
			resWant: func(t *testing.T, res *http.Response) {
				assert.Equal(t, 200, res.StatusCode)
				payload, err := io.ReadAll(res.Body)
				assert.NoError(t, err)
				assert.Equal(t, "{\"errors\":[{\"message\":\"PersistedOperationNotFound\"}]}\n", string(payload))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := slog.Default()
			po, _ := NewPersistedOperations(log, tt.args.cfg, newMemoryLoader(tt.args.cache), nil)
			po.cache = tt.args.cache

			req := httptest.NewRequest("POST", "/", bytes.NewBuffer(tt.args.payload))
			resp := httptest.NewRecorder()

			po.SwapHashForQuery(tt.want(t)).ServeHTTP(resp, req)
			res := resp.Result()
			defer res.Body.Close()

			tt.resWant(t, res)
		})
	}
}
