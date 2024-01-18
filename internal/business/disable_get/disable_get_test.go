package disable_get

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDisableMethodRule(t *testing.T) {
	type args struct {
		cfg     Config
		request *http.Request
	}
	tests := []struct {
		name string
		args args
		want func(res *http.Response)
	}{
		{
			name: "does not block when disabled",
			args: args{
				cfg: Config{
					Enabled: false,
				},
				request: func() *http.Request {
					return httptest.NewRequest("GET", "/graphql", nil)
				}(),
			},
			want: func(res *http.Response) {
				assert.Equal(t, 200, res.StatusCode)
			},
		},
		{
			name: "does not block GETs that contain no operation",
			args: args{
				cfg: Config{
					Enabled: true,
				},
				request: func() *http.Request {
					return httptest.NewRequest("GET", "/graphql", nil)
				}(),
			},
			want: func(res *http.Response) {
				assert.Equal(t, 200, res.StatusCode)
			},
		},
		{
			name: "does block GETs that contain an operation",
			args: args{
				cfg: Config{
					Enabled: true,
				},
				request: func() *http.Request {
					return httptest.NewRequest("GET", "/graphql?query=foobar&variables=something", nil)
				}(),
			},
			want: func(res *http.Response) {
				assert.Equal(t, http.StatusMethodNotAllowed, res.StatusCode)
			},
		},
		{
			name: "does block GETs that contain extensions for blocking persisted operations",
			args: args{
				cfg: Config{
					Enabled: true,
				},
				request: func() *http.Request {
					return httptest.NewRequest("GET", "/graphql?extensions=something", nil)
				}(),
			},
			want: func(res *http.Response) {
				assert.Equal(t, http.StatusMethodNotAllowed, res.StatusCode)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			mw := DisableMethodRule(tt.args.cfg)
			mw(requestRecorder{}).ServeHTTP(rec, tt.args.request)
			result := rec.Result()
			defer result.Body.Close()

			tt.want(result)
		})
	}
}

type requestRecorder struct {
}

func (r requestRecorder) ServeHTTP(writer http.ResponseWriter, _ *http.Request) {
	_, _ = writer.Write([]byte("hello world"))
}
