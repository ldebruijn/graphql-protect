package readiness

import (
	"fmt"
	"net/http"
)

func NewReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "I'm ready!")
	}
}
