package main

import (
	"bytes"
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
	"time"
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
						"query": "query Foo { product(id: 1) { id name } }",
					}

					bts, _ := json.Marshal(body)
					r := httptest.NewRequest("POST", "/graphql", bytes.NewBuffer(bts))
					return r
				}(),
				cfgOverrides: func(cfg *config.Config) *config.Config {
					cfg.PersistedOperations.Enabled = true
					cfg.PersistedOperations.Store = "./"
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
				actual, err := io.ReadAll(response.Body)
				assert.NoError(t, err)
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
					cfg.PersistedOperations.Store = "./"
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
				actual, err := io.ReadAll(response.Body)
				assert.NoError(t, err)

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
						"query": "query Foo { product(id: 1) { id name } }",
					}

					bts, _ := json.Marshal(body)
					r := httptest.NewRequest("POST", "/graphql", bytes.NewBuffer(bts))
					return r
				}(),
				cfgOverrides: func(cfg *config.Config) *config.Config {
					cfg.PersistedOperations.Enabled = true
					cfg.PersistedOperations.Store = "./"
					cfg.PersistedOperations.FailUnknownOperations = false
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
				actual, err := io.ReadAll(response.Body)
				assert.NoError(t, err)
				// perform string comparisons as map[string]interface seems incomparable
				assert.Equal(t, string(ex), string(actual))
			},
		},
		{
			name: "blocks requests with too many aliases",
			args: args{
				request: func() *http.Request {
					body := map[string]interface{}{
						"query": `
query Foo { 
	a1: uploadImage(image: $image)
	a2: uploadImage(image: $image)
	a3: uploadImage(image: $image)
	a4: uploadImage(image: $image)
	a5: uploadImage(image: $image)
	a6: uploadImage(image: $image)
	a7: uploadImage(image: $image)
	a8: uploadImage(image: $image)
	a9: uploadImage(image: $image)
	a10: uploadImage(image: $image)
}`,
					}

					bts, _ := json.Marshal(body)
					r := httptest.NewRequest("POST", "/graphql", bytes.NewBuffer(bts))
					return r
				}(),
				cfgOverrides: func(cfg *config.Config) *config.Config {
					cfg.MaxAliases.Enabled = true
					cfg.MaxAliases.Max = 3
					return cfg
				},
				mockResponse: map[string]interface{}{
					"data": map[string]interface{}{
						"a1":  "Yes",
						"a2":  "Yes",
						"a3":  "Yes",
						"a4":  "Yes",
						"a5":  "Yes",
						"a6":  "Yes",
						"a7":  "Yes",
						"a8":  "Yes",
						"a9":  "Yes",
						"a10": "Yes",
					},
				},
			},
			want: func(t *testing.T, response *http.Response) {
				expected := map[string]interface{}{
					"errors": []map[string]interface{}{
						{
							"message": "syntax error: Aliases limit of 3 exceeded, found 10",
						},
					},
				}
				_, _ = json.Marshal(expected)
				actual, err := io.ReadAll(response.Body)
				assert.NoError(t, err)
				// perform string comparisons as map[string]interface seems incomparable
				assert.True(t, errorsContainsMessage("syntax error: Aliases limit of 3 exceeded, found 10", actual))
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

			defaultConfig, _ := config.NewConfig("")
			cfg := tt.args.cfgOverrides(defaultConfig)

			// set target to mockserver
			cfg.Target.Host = mockServer.URL

			go func() {
				_ = run(slog.Default(), cfg, shutdown)
			}()

			// tiny sleep to make sure HTTP server has started
			time.Sleep(100 * time.Millisecond)

			url := "http://localhost:8080" + tt.args.request.URL.String()
			res, err := http.Post(url, tt.args.request.Header.Get("Content-Type"), tt.args.request.Body)
			if err != nil {
				assert.NoError(t, err, tt.name)
			}

			tt.want(t, res)

			// cleanup
			shutdown <- syscall.SIGINT
		})
	}
}

func errorsContainsMessage(msg string, bytes []byte) bool {
	var payload map[string]interface{}
	err := json.Unmarshal(bytes, &payload)
	if err != nil {
		return false
	}

	if errors, ok := payload["errors"]; ok {
		for _, err := range errors.([]interface{}) {
			if errMsg, ok := err.(map[string]interface{})["message"]; ok {
				if msg == errMsg {
					return true
				}
			}
		}
	}
	return false
}
