package accesslogging

import (
	"context"

	"github.com/ldebruijn/graphql-protect/internal/business/gql"
)

// LogWriter defines the interface for writing access logs to different backends
type LogWriter interface {
	// Write writes log entries to the backend
	Write(data LogEntryData)

	// Shutdown performs cleanup and flushes any pending logs
	Shutdown(ctx context.Context) error
}

// LogEntryData represents structured data for a single access log entry
type LogEntryData struct {
	Payloads        []gql.RequestData
	FilteredHeaders map[string]interface{} // Pre-filtered headers
	IncludeOpName   bool
	IncludeVars     bool
	IncludePayload  bool
}

// buildAccessLogs creates accessLog entries from the provided data
func buildAccessLogs(data LogEntryData) []accessLog {
	logs := make([]accessLog, 0, len(data.Payloads))
	for _, req := range data.Payloads {
		al := accessLog{}

		if data.IncludeOpName {
			al.WithOperationName(req.OperationName)
		}
		if data.IncludeVars {
			al.WithVariables(req.Variables)
		}
		if data.IncludePayload {
			al.WithPayload(req.Query)
		}

		al.WithHeaders(data.FilteredHeaders)
		logs = append(logs, al)
	}
	return logs
}
