package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ldebruijn/go-graphql-armor/internal/app/config"
	"github.com/stretchr/testify/assert"
	"io"
	log2 "log"
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
		request        *http.Request
		mockResponse   map[string]interface{}
		mockStatusCode int
		cfgOverrides   func(cfg *config.Config) *config.Config
		schema         string
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
				schema: `
extend type Query {
	product(id: ID!): Product
}

type Product {
	id: ID!
	name: String
}
`,
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
				mockStatusCode: http.StatusOK,
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
				schema: `
extend type Query {
	product(id: ID!): Product
}

type Product {
	id: ID!
	name: String
}
`,
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
				mockStatusCode: http.StatusOK,
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
				schema: `
extend type Query {
	product(id: ID!): Product
}

type Product {
	id: ID!
	name: String
}
`,
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
				mockStatusCode: http.StatusOK,
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
query Foo($image: ImageInput!) { 
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
						"variables": map[string]interface{}{
							"image": map[string]interface{}{
								"id": "1",
							},
						},
					}

					bts, _ := json.Marshal(body)
					r := httptest.NewRequest("POST", "/graphql", bytes.NewBuffer(bts))
					return r
				}(),
				schema: `
extend type Query {
	uploadImage(image: ImageInput!): String
}

input ImageInput {
	id: ID!
}
`,
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
				mockStatusCode: http.StatusOK,
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
		{
			name: "redacts error message of request with too many aliases",
			args: args{
				request: func() *http.Request {
					body := map[string]interface{}{
						"query": `
query Foo($image: ImageInput!) { 
	a1: uploadImage(image: $image)
	a2: uploadImage(image: $image)
}`,
						"variables": map[string]interface{}{
							"image": map[string]interface{}{
								"id": "1",
							},
						},
					}

					bts, _ := json.Marshal(body)
					r := httptest.NewRequest("POST", "/graphql", bytes.NewBuffer(bts))
					return r
				}(),
				schema: `
extend type Query {
	uploadImage(image: ImageInput!): String
}

input ImageInput {
	id: ID!
}
`,
				cfgOverrides: func(cfg *config.Config) *config.Config {
					cfg.MaxAliases.Enabled = true
					cfg.MaxAliases.Max = 1
					cfg.ObfuscateValidationErrors = true
					return cfg
				},
				mockResponse: map[string]interface{}{
					"data": map[string]interface{}{
						"a1": "Yes",
						"a2": "Yes",
					},
				},
				mockStatusCode: http.StatusOK,
			},
			want: func(t *testing.T, response *http.Response) {
				expected := map[string]interface{}{
					"errors": []map[string]interface{}{
						{
							"message": errRedacted.Error(),
						},
					},
				}
				_, _ = json.Marshal(expected)
				actual, err := io.ReadAll(response.Body)
				assert.NoError(t, err)
				// perform string comparisons as map[string]interface seems incomparable
				fmt.Println(string(actual))
				assert.True(t, errorsContainsMessage(errRedacted.Error(), actual))
			},
		},
		{
			name: "correctly handles unexpected response when response mutators are enabled",
			args: args{
				request: func() *http.Request {
					body := map[string]interface{}{
						"query": "query Foo { product(id: 1) { id name } }",
					}

					bts, _ := json.Marshal(body)
					r := httptest.NewRequest("POST", "/graphql", bytes.NewBuffer(bts))
					return r
				}(),
				schema: `
extend type Query {
	product(id: ID!): Product
}

type Product {
	id: ID!
	name: String
}
`,
				cfgOverrides: func(cfg *config.Config) *config.Config {
					cfg.PersistedOperations.Enabled = true
					cfg.PersistedOperations.Store = "./"
					cfg.PersistedOperations.FailUnknownOperations = false
					return cfg
				},
				mockResponse: map[string]interface{}{
					"data": map[string]interface{}{
						"product": nil,
					},
					"errors": []map[string]interface{}{
						{
							"message": "Some unexpected error",
							"path":    []string{"product"},
							"extensions": map[string]interface{}{
								"errorType": "BAD_REQUEST",
							},
						},
					},
				},
				mockStatusCode: http.StatusBadRequest,
			},
			want: func(t *testing.T, response *http.Response) {
				expected := map[string]interface{}{
					"errors": []map[string]interface{}{
						{
							"message": "Some unexpected error",
							"path":    []string{"product"},
							"extensions": map[string]interface{}{
								"errorType": "BAD_REQUEST",
							},
						},
					},
				}
				_, _ = json.Marshal(expected)
				assert.Equal(t, http.StatusBadRequest, response.StatusCode)
				actual, err := io.ReadAll(response.Body)
				assert.NoError(t, err)
				// perform string comparisons as map[string]interface seems incomparable
				assert.True(t, errorsContainsMessage("Some unexpected error", actual))
			},
		},
		{
			name: "validates incoming request payload against schema",
			args: args{
				request: func() *http.Request {
					body := map[string]interface{}{
						"query": "query Foo($id: ID!) { product(id: $id) { id name } }",
					}

					bts, _ := json.Marshal(body)
					r := httptest.NewRequest("POST", "/graphql", bytes.NewBuffer(bts))
					return r
				}(),
				schema: `
extend type Query {
	product(id: ID!): Product
}

type Product {
	id: ID!
	name: String
}
`,
				cfgOverrides: func(cfg *config.Config) *config.Config {
					cfg.PersistedOperations.Enabled = true
					cfg.PersistedOperations.Store = "./"
					cfg.PersistedOperations.FailUnknownOperations = false
					return cfg
				},
				mockResponse: map[string]interface{}{
					"data": map[string]interface{}{
						"product": map[string]interface{}{
							"id":   "1",
							"name": "name",
						},
					},
				},
				mockStatusCode: http.StatusOK,
			},
			want: func(t *testing.T, response *http.Response) {
				expected := map[string]interface{}{
					"data": map[string]interface{}{
						"product": map[string]interface{}{
							"id":   "1",
							"name": "name",
						},
					},
				}
				_, _ = json.Marshal(expected)
				assert.Equal(t, http.StatusOK, response.StatusCode)
				actual, err := io.ReadAll(response.Body)
				assert.NoError(t, err)
				_ = actual
				assert.NotContains(t, string(actual), "\"errors\":")
			},
		},
		{
			name: "throws error when exceeding max tokens",
			args: args{
				request: func() *http.Request {
					body := map[string]interface{}{
						"query": "query Foo($id: ID!) { product(id: $id) { id name } }",
					}

					bts, _ := json.Marshal(body)
					r := httptest.NewRequest("POST", "/graphql", bytes.NewBuffer(bts))
					return r
				}(),
				schema: `
extend type Query {
	product(id: ID!): Product
}

type Product {
	id: ID!
	name: String
}
`,
				cfgOverrides: func(cfg *config.Config) *config.Config {
					cfg.MaxTokens.Enabled = true
					cfg.MaxTokens.Max = 1
					return cfg
				},
				mockResponse: map[string]interface{}{
					"data": map[string]interface{}{
						"product": map[string]interface{}{
							"id":   "1",
							"name": "name",
						},
					},
				},
				mockStatusCode: http.StatusOK,
			},
			want: func(t *testing.T, response *http.Response) {
				assert.Equal(t, http.StatusOK, response.StatusCode)
				actual, err := io.ReadAll(response.Body)
				assert.NoError(t, err)
				_ = actual
				assert.Contains(t, string(actual), "operation has exceeded maximum tokens. found [22], max [1]")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				bts, _ := json.Marshal(tt.args.mockResponse)
				w.WriteHeader(tt.args.mockStatusCode)

				_, _ = w.Write(bts)
			}))
			defer mockServer.Close()

			shutdown := make(chan os.Signal, 1)

			// create temp file for storing schema
			file, _ := os.CreateTemp("", "")
			defer func() {
				_ = os.Remove(file.Name())
			}()

			write, err := file.Write([]byte(tt.args.schema))
			assert.NoError(t, err)
			assert.NotEqual(t, 0, write)
			_ = file.Close()

			defaultConfig, _ := config.NewConfig("")
			cfg := tt.args.cfgOverrides(defaultConfig)

			// set target to mockserver
			cfg.Target.Host = mockServer.URL
			cfg.Schema.Path = file.Name()

			go func() {
				err := run(slog.Default(), cfg, shutdown)
				assert.NoError(t, err, "error starting server for", tt.name)
			}()

			start := time.Now()

			// block test case until server has started
			log2.Println("Waiting until server has started")

			blockUntilStarted(httptest.NewRequest("GET", "/", nil), 1*time.Second)

			log2.Printf("Server has started, took %s \n", time.Since(start))

			url := "http://localhost:8080" + tt.args.request.URL.String()
			res, err := http.Post(url, tt.args.request.Header.Get("Content-Type"), tt.args.request.Body)

			assert.NoError(t, err, tt.name)
			assert.NotNil(t, res, "response was nil", tt.name)

			tt.want(t, res)

			// cleanup
			shutdown <- syscall.SIGINT

			// give time for server to shutdown
			log2.Println("Waiting until server has shut down")
			time.Sleep(100 * time.Millisecond)
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

func blockUntilStarted(req *http.Request, timeout time.Duration) {
	client := &http.Client{
		Timeout: 100 * time.Millisecond,
	}
	for start := time.Now(); time.Since(start) < timeout; {
		resp, err := client.Do(req)
		if err != nil {
			// tiny sleep before next iteration
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if resp.StatusCode == http.StatusOK {
			return
		}
	}
}
