package trusteddocuments // nolint:revive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
					"foobar": newPersistedOperation("query { foobar }"),
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
					"foobar": newPersistedOperation("query { foobar }"),
					"baz":    newPersistedOperation("query { baz }"),
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
					"foobar": newPersistedOperation("query { foobar }")},
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
			po, _ := NewPersistedOperations(log, tt.args.cfg, newMemoryLoader(tt.args.cache))
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

func TestLoader(t *testing.T) {
	type args struct {
		state           map[string]PersistedOperation
		loader          Loader
		failureStrategy ReloadFailureStrategy
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]PersistedOperation
		wantErr error
	}{
		{
			name: "loader state is added to cache",
			args: args{
				loader: func() Loader {
					data := map[string]PersistedOperation{}
					data["123"] = PersistedOperation{
						Operation: "i am an operation",
						Name:      "i am a name",
					}

					loader := newMemoryLoader(data)

					return loader
				}(),
				state:           map[string]PersistedOperation{},
				failureStrategy: ReloadFailureStrategyReject,
			},
			want: map[string]PersistedOperation{
				"123": {
					Operation: "i am an operation",
					Name:      "i am a name",
				},
			},
			wantErr: nil,
		},
		{
			name: "loader state overwrites cache, does not append",
			args: args{
				loader: func() Loader {
					data := map[string]PersistedOperation{}
					data["123"] = PersistedOperation{
						Operation: "i am an operation",
						Name:      "i am a name",
					}

					loader := newMemoryLoader(data)

					return loader
				}(),
				state: map[string]PersistedOperation{
					"456": {
						Operation: "i am an operation that does get deleted",
						Name:      "i am a name that doest get deleted",
					},
				},
				failureStrategy: ReloadFailureStrategyReject,
			},
			want: map[string]PersistedOperation{
				"123": {
					Operation: "i am an operation",
					Name:      "i am a name",
				},
			},
			wantErr: nil,
		},
		{
			name: "loader error does not update cache with failure mode reject",
			args: args{
				loader: func() Loader {
					loader := &testLoader{
						err:             errors.New("this is unexpected"),
						willReturnError: false,
					}

					return loader
				}(),
				state: map[string]PersistedOperation{
					"456": {
						Operation: "i am an operation that does not get deleted",
						Name:      "i am a name that doest not get deleted",
					},
				},
				failureStrategy: ReloadFailureStrategyReject,
			},
			want: map[string]PersistedOperation{
				"456": {
					Operation: "i am an operation that does not get deleted",
					Name:      "i am a name that doest not get deleted",
				},
			},
			wantErr: errors.New("this is unexpected"),
		},
		{
			name: "loader error does update cache with failure mode ignore",
			args: args{
				loader: func() Loader {
					loader := &testLoader{
						err: errors.New("this is unexpected"),
						data: map[string]PersistedOperation{
							"123": {
								Operation: "i am an operation",
								Name:      "i am a name",
							},
						},
						willReturnError: false,
					}

					return loader
				}(),
				state: map[string]PersistedOperation{
					"456": {
						Operation: "i am an operation that does not get deleted",
						Name:      "i am a name that doest not get deleted",
					},
				},
				failureStrategy: ReloadFailureStrategyIgnore,
			},
			want: map[string]PersistedOperation{
				"123": {
					Operation: "i am an operation",
					Name:      "i am a name",
				},
			},
			wantErr: errors.New("this is unexpected"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := slog.Default()
			po, _ := NewPersistedOperations(log, Config{}, tt.args.loader)
			po.cache = tt.args.state

			err := po.load(tt.args.failureStrategy)
			if tt.wantErr != nil {
				assert.Error(t, tt.wantErr, err)
			}

			assert.Equal(t, tt.want, po.cache)
		})
	}
}

var _ Loader = &testLoader{}

// ErrorLoader is a loader for testing purposes
type testLoader struct {
	data            map[string]PersistedOperation
	err             error
	willReturnError bool
}

func (e *testLoader) Type() string {
	return "error"
}

func newPersistedOperation(query string) PersistedOperation {
	return PersistedOperation{query, extractOperationNameFromOperation(query)}
}

func (e *testLoader) Load(_ context.Context) (map[string]PersistedOperation, error) {
	if e.willReturnError {
		return e.data, e.err
	}
	// return error after the first call
	e.willReturnError = true

	return e.data, nil
}
