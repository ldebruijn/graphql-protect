package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/ldebruijn/go-graphql-armor/internal/app/config"
	"github.com/stretchr/testify/assert"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"syscall"
	"testing"
)

func TestHttpServerIntegration(t *testing.T) {
	type args struct {
		request      *http.Request
		mockResponse map[string]interface{}
		cfgOverrides func(cfg *config.Config) *config.Config
	}
	tests := []struct {
		name string
		args args
		want func(t *testing.T, response *http.Response)
	}{
		{
			name: "regular request yields regular response",
			args: args{
				request: func() *http.Request {
					body := map[string]interface{}{
						"query": "query { product(id: 1) { id name } }",
					}

					bts, _ := json.Marshal(body)
					r := httptest.NewRequest("POST", "/graphql", bytes.NewBuffer(bts))
					return r
				}(),
				cfgOverrides: func(cfg *config.Config) *config.Config {
					cfg.PersistedOperations.Enabled = true
					return cfg
				},
				mockResponse: map[string]interface{}{
					"data": map[string]interface{}{
						"product": map[string]interface{}{
							"id":   1,
							"name": "hi",
						},
					},
				},
			},
			want: func(t *testing.T, response *http.Response) {
				expected := map[string]interface{}{
					"data": map[string]interface{}{
						"product": map[string]interface{}{
							"id":   1,
							"name": "hi",
						},
					},
				}
				ex, _ := json.Marshal(expected)
				actual, _ := io.ReadAll(response.Body)
				// perform string comparisons as map[string]interface seems incomparable
				assert.Equal(t, string(ex), string(actual))
			},
		},
		{
			name: "blocks requests on unknown persisted operations",
			args: args{
				request: func() *http.Request {
					body := map[string]interface{}{
						"extensions": map[string]interface{}{
							"persistedQuery": map[string]interface{}{
								"sha256Hash": "foobar",
							},
						},
					}

					bts, _ := json.Marshal(body)
					r := httptest.NewRequest("POST", "/graphql", bytes.NewBuffer(bts))
					return r
				}(),
				cfgOverrides: func(cfg *config.Config) *config.Config {
					cfg.PersistedOperations.Enabled = true
					return cfg
				},
				mockResponse: map[string]interface{}{
					"data": map[string]interface{}{
						"product": map[string]interface{}{
							"id":   1,
							"name": "hi",
						},
					},
				},
			},
			want: func(t *testing.T, response *http.Response) {
				expected := map[string]interface{}{
					"errors": []map[string]interface{}{
						{
							"message": "PersistedOperationNotFound",
						},
					},
				}
				ex, _ := json.Marshal(expected)
				actual, _ := io.ReadAll(response.Body)
				ac := string(actual)
				ac = strings.TrimSuffix(ac, "\n")

				// perform string comparisons as map[string]interface seems incomparable
				assert.Equal(t, string(ex), ac)
			},
		},
		{
			name: "removes field suggestions from response",
			args: args{
				request: func() *http.Request {
					body := map[string]interface{}{
						"query": "query { product(id: 1) { id name } }",
					}

					bts, _ := json.Marshal(body)
					r := httptest.NewRequest("POST", "/graphql", bytes.NewBuffer(bts))
					return r
				}(),
				cfgOverrides: func(cfg *config.Config) *config.Config {
					cfg.PersistedOperations.Enabled = true
					return cfg
				},
				mockResponse: map[string]interface{}{
					"data": map[string]interface{}{
						"product": map[string]interface{}{
							"id":   1,
							"name": "hi",
						},
					},
					"errors": []map[string]interface{}{
						{
							"message":        "Did you mean something else?",
							"something else": "lol",
						},
					},
				},
			},
			want: func(t *testing.T, response *http.Response) {
				expected := map[string]interface{}{
					"data": map[string]interface{}{
						"product": map[string]interface{}{
							"id":   1,
							"name": "hi",
						},
					},
					"errors": []map[string]interface{}{
						{
							"message":        "[redacted]",
							"something else": "lol",
						},
					},
				}
				ex, _ := json.Marshal(expected)
				actual, _ := io.ReadAll(response.Body)
				// perform string comparisons as map[string]interface seems incomparable
				assert.Equal(t, string(ex), string(actual))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				bts, _ := json.Marshal(tt.args.mockResponse)

				_, _ = w.Write(bts)
			}))
			defer mockServer.Close()

			shutdown := make(chan os.Signal, 1)

			defaultConfig, _ := config.NewConfig()
			cfg := tt.args.cfgOverrides(defaultConfig)

			// set target to mockserver
			cfg.Target.Host = mockServer.URL

			go func() {
				_ = run(context.Background(), slog.Default(), cfg, shutdown)
			}()

			url := "http://localhost:8080" + tt.args.request.URL.String()
			res, err := http.Post(url, tt.args.request.Header.Get("Content-Type"), tt.args.request.Body)
			if err != nil {
				assert.NoError(t, err)
			}

			tt.want(t, res)

			// cleanup
			shutdown <- syscall.SIGINT
		})
	}
}
