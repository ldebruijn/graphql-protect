package debug

import (
	"encoding/json"
	"github.com/ldebruijn/graphql-protect/internal/business/persistedoperations"
	"net/http"
)

func NewTrustedDocumentsDebugger(po *persistedoperations.Handler, enableDebugEndpoint bool) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if enableDebugEndpoint {

			trustedDocuments := po.GetTrustedDocuments()

			jsonData, err := json.MarshalIndent(trustedDocuments, "", "  ")
			if err != nil {
				return
			}

			// Set the content type to application/json
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			// Write the JSON response
			_, _ = w.Write(jsonData)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}
}
