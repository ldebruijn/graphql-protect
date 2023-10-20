package readiness

import (
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewReadinessHandler(t *testing.T) {
	tests := []struct {
		name string
		want func(t *testing.T, res *http.Response)
	}{
		{
			name: "Sends a 200 status code with an \"I'm ready!\" body",
			want: func(t *testing.T, res *http.Response) {
				assert.Equal(t, http.StatusOK, res.StatusCode)
				payload, _ := io.ReadAll(res.Body)
				assert.Equal(t, "I'm ready!", string(payload))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)

			handler := NewReadinessHandler()
			resp := httptest.NewRecorder()

			handler.ServeHTTP(resp, req)

			tt.want(t, resp.Result())
		})
	}
}
