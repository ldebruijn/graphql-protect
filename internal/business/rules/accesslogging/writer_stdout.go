package accesslogging

import (
	"context"
	"log/slog"
)

var _ LogWriter = &StdoutWriter{}

// StdoutWriter writes access logs to stdout using slog
type StdoutWriter struct {
	log *slog.Logger
}

// NewStdoutWriter creates a new stdout log writer
func NewStdoutWriter(log *slog.Logger) *StdoutWriter {
	return &StdoutWriter{
		log: log.WithGroup("access-logging"),
	}
}

// Write writes a log entry to stdout
func (w *StdoutWriter) Write(data LogEntryData) {
	for _, al := range buildAccessLogs(data) {
		w.log.Info("record", "payload", al)
	}
}

// Shutdown is a no-op for stdout writer
func (w *StdoutWriter) Shutdown(_ context.Context) error {
	return nil
}
