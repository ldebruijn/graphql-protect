package proxy

import (
	"github.com/ldebruijn/graphql-protect/internal/business/rules/block_field_suggestions"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func Test_modifyResponse(t *testing.T) {
	type args struct {
		blockFieldSuggestions *block_field_suggestions.BlockFieldSuggestionsHandler
		response              *http.Response
	}
	tests := []struct {
		name string
		args args
		want func(res *http.Response)
	}{
		{
			name: "nothing if disabled",
			args: args{
				blockFieldSuggestions: func() *block_field_suggestions.BlockFieldSuggestionsHandler {
					return block_field_suggestions.NewBlockFieldSuggestionsHandler(block_field_suggestions.Config{
						Enabled: false,
					})
				}(),
				response: func() *http.Response {
					return &http.Response{
						Status:        "200",
						StatusCode:    200,
						Body:          io.NopCloser(strings.NewReader("this is not valid json")),
						Proto:         "HTTP/1.1",
						ProtoMajor:    1,
						ProtoMinor:    1,
						ContentLength: 0,
						Header:        map[string][]string{},
					}
				}(), // nolint:bodyclose
			},
			want: func(res *http.Response) {
				body, _ := io.ReadAll(res.Body)
				assert.Equal(t, 200, res.StatusCode)
				assert.Equal(t, "200", res.Status)
				assert.Equal(t, "this is not valid json", string(body))
			},
		},
		{
			name: "handles non-json gracefully",
			args: args{
				blockFieldSuggestions: func() *block_field_suggestions.BlockFieldSuggestionsHandler {
					return block_field_suggestions.NewBlockFieldSuggestionsHandler(block_field_suggestions.Config{
						Enabled: true,
					})
				}(),
				response: func() *http.Response {
					return &http.Response{
						Status:        "200",
						StatusCode:    200,
						Body:          io.NopCloser(strings.NewReader("this is not valid json")),
						Proto:         "HTTP/1.1",
						ProtoMajor:    1,
						ProtoMinor:    1,
						ContentLength: 0,
						Header:        map[string][]string{},
					}
				}(), // nolint:bodyclose
			},
			want: func(res *http.Response) {
				body, _ := io.ReadAll(res.Body)
				assert.Equal(t, 200, res.StatusCode)
				assert.Equal(t, "200", res.Status)
				assert.Equal(t, "this is not valid json", string(body))
			},
		},
		{
			name: "handles invalid-json gracefully",
			args: args{
				blockFieldSuggestions: func() *block_field_suggestions.BlockFieldSuggestionsHandler {
					return block_field_suggestions.NewBlockFieldSuggestionsHandler(block_field_suggestions.Config{
						Enabled: true,
					})
				}(),
				response: func() *http.Response { // nolint:bodyclose
					return &http.Response{
						Status:        "200",
						StatusCode:    200,
						Body:          io.NopCloser(strings.NewReader("{ \"this\": \" is not valid json }")),
						Proto:         "HTTP/1.1",
						ProtoMajor:    1,
						ProtoMinor:    1,
						ContentLength: 0,
						Header:        map[string][]string{},
					}
				}(), // nolint:bodyclose
			},
			want: func(res *http.Response) {
				body, _ := io.ReadAll(res.Body)
				assert.Equal(t, 200, res.StatusCode)
				assert.Equal(t, "200", res.Status)
				assert.Equal(t, "{ \"this\": \" is not valid json }", string(body))
			},
		},
		{
			name: "handles json gracefully",
			args: args{
				blockFieldSuggestions: func() *block_field_suggestions.BlockFieldSuggestionsHandler {
					return block_field_suggestions.NewBlockFieldSuggestionsHandler(block_field_suggestions.Config{
						Enabled: true,
						Mask:    "[masked]",
					})
				}(),
				response: func() *http.Response {
					return &http.Response{
						Status:        "200",
						StatusCode:    200,
						Body:          io.NopCloser(strings.NewReader("{ \"errors\": [{\"message\": \"Did you mean \"}] }")),
						Proto:         "HTTP/1.1",
						ProtoMajor:    1,
						ProtoMinor:    1,
						ContentLength: 0,
						Header:        map[string][]string{},
					}
				}(), // nolint:bodyclose
			},
			want: func(res *http.Response) {
				body, _ := io.ReadAll(res.Body)
				assert.Equal(t, 200, res.StatusCode)
				assert.Equal(t, "200", res.Status)
				assert.Equal(t, "{\"errors\":[{\"message\":\"[masked]\"}]}", string(body))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			result := modifyResponse(tt.args.blockFieldSuggestions, nil) // nolint:bodyclose

			_ = result(tt.args.response)
			tt.want(tt.args.response)
		})
	}
}

func TestForwardsXff(t *testing.T) {
	rr := &RequestRecorder{}
	testServer := httptest.NewServer(rr)
	upstreamURL, err := url.Parse(testServer.URL)
	assert.NoError(t, err)

	cfg := Config{
		Timeout:   1 * time.Second,
		KeepAlive: 180 * time.Second,
		Host:      "http://" + upstreamURL.Host,
		Tracing:   TracingConfig{},
	}
	proxy, err := NewProxy(cfg, nil, nil)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("x-forwarded-for", "123.456.789.0")

	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	rr.Assert(func(r *http.Request) {
		val := r.Header.Get("x-forwarded-for")
		assert.True(t, strings.HasPrefix(val, "123.456.789.0,")) // trailing , to make sure IP of protect is appended to list
	})
}

type RequestRecorder struct {
	lastRequest *http.Request
}

func (r *RequestRecorder) ServeHTTP(_ http.ResponseWriter, request *http.Request) {
	r.lastRequest = request
}

func (r *RequestRecorder) Assert(assert func(r *http.Request)) {
	assert(r.lastRequest)
}
