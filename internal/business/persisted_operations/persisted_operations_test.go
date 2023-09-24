package persisted_operations

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewPersistedOperations(t *testing.T) {
	type args struct {
		cfg     Config
		payload RequestPayload
		cache   map[string]string
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
			want: func(t *testing.T) http.Handler {
				fn := func(w http.ResponseWriter, r *http.Request) {
				}
				return http.HandlerFunc(fn)
			},
			resWant: func(t *testing.T, res *http.Response) {

			},
		},
		{
			name: "Allows unpersisted requests if configured",
			args: args{
				cfg: Config{
					Enabled:                    true,
					AllowUnPersistedOperations: true,
				},
				payload: RequestPayload{
					Query: "query { foo }",
				},
			},
			want: func(t *testing.T) http.Handler {
				fn := func(w http.ResponseWriter, r *http.Request) {
					decoder := json.NewDecoder(r.Body)

					var payload RequestPayload
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
					Enabled:                    true,
					AllowUnPersistedOperations: false,
				},
				payload: RequestPayload{
					Extensions: Extensions{
						PersistedQuery: &PersistedQuery{
							Sha256Hash: "foobar",
						},
					},
				},
				cache: map[string]string{},
			},
			want: func(t *testing.T) http.Handler {
				fn := func(w http.ResponseWriter, r *http.Request) {

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
					Enabled:                    true,
					AllowUnPersistedOperations: false,
				},
				payload: RequestPayload{
					Extensions: Extensions{
						PersistedQuery: &PersistedQuery{
							Sha256Hash: "foobar",
						},
					},
				},
				cache: map[string]string{
					"foobar": "query { foobar }",
				},
			},
			want: func(t *testing.T) http.Handler {
				fn := func(w http.ResponseWriter, r *http.Request) {
					decoder := json.NewDecoder(r.Body)

					var payload RequestPayload
					err := decoder.Decode(&payload)
					assert.NoError(t, err)

					assert.Equal(t, "query { foobar }", payload.Query)
					assert.Equal(t, int64(82), r.ContentLength)

					length, _ := json.Marshal(payload)
					assert.Equal(t, 82, len(length))
				}
				return http.HandlerFunc(fn)
			},
			resWant: func(t *testing.T, res *http.Response) {
				assert.Equal(t, 200, res.StatusCode)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := slog.Default()
			po, _ := NewPersistedOperations(log, tt.args.cfg, newMemoryLoader(tt.args.cache))
			po.cache = tt.args.cache

			bts, err := json.Marshal(&tt.args.payload)
			if err != nil {
				assert.NoError(t, err)
				return
			}

			req := httptest.NewRequest("POST", "/", bytes.NewBuffer(bts))
			resp := httptest.NewRecorder()

			po.Execute(tt.want(t)).ServeHTTP(resp, req)
			tt.resWant(t, resp.Result())
		})
	}
}
