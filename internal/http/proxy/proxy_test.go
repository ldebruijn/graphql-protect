package proxy

import (
	"github.com/ldebruijn/graphql-protect/internal/business/rules/block_field_suggestions"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"strings"
	"testing"
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
		t.Run(tt.name, func(t *testing.T) {
			result := modifyResponse(tt.args.blockFieldSuggestions) // nolint:bodyclose

			_ = result(tt.args.response)
			tt.want(tt.args.response)
		})
	}
}
