package readiness

import (
	"fmt"
	"github.com/ldebruijn/graphql-protect/internal/business/persistedoperations"
	"net/http"
)

func NewReadinessHandler(po *persistedoperations.Handler) http.HandlerFunc {

	if po != nil &&
		po.Config().Enabled && po.Config().RejectOnFailure &&
		po.PersistedOpsInMemory() == 0 {
		return func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = fmt.Fprintf(w, "Not ready yet!")
		}
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "I'm ready!")
	}
}
