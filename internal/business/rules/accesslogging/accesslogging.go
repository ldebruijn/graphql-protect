package accesslogging

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/ldebruijn/graphql-protect/internal/business/gql"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	droppedLogsCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "graphql_protect",
		Subsystem: "access_logging",
		Name:      "dropped_logs_total",
		Help:      "The total number of access log entries dropped due to full channel buffer",
	})

	bufferUsageGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "graphql_protect",
		Subsystem: "access_logging",
		Name:      "buffer_usage_current",
		Help:      "The current number of log entries in the async buffer",
	})

	bufferSizeGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "graphql_protect",
		Subsystem: "access_logging",
		Name:      "buffer_size_limit",
		Help:      "The maximum capacity of the async logging buffer",
	})
)

func init() {
	prometheus.MustRegister(droppedLogsCounter, bufferUsageGauge, bufferSizeGauge)
}

type logEntry struct {
	payloads []gql.RequestData
	headers  http.Header
}

type AccessLogging struct {
	log                  *slog.Logger
	enabled              bool
	includeHeaders       map[string]bool
	includeOperationName bool
	includeVariables     bool
	includePayload       bool
	async                bool
	writer               LogWriter
	logChan              chan logEntry
	shutdown             chan struct{}
	wg                   sync.WaitGroup
}

func NewAccessLogging(cfg Config, log *slog.Logger) (*AccessLogging, error) {
	headers := map[string]bool{}
	for _, header := range cfg.IncludedHeaders {
		headers[header] = true
	}

	// Select writer based on configuration
	var writer LogWriter
	var err error

	if cfg.GoogleCloudLogging.Enabled {
		// Use Google Cloud Logging writer
		writer, err = NewGoogleCloudWriter(cfg.GoogleCloudLogging, log)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Google Cloud Logging writer: %w", err)
		}
	} else {
		// Use stdout writer (default)
		writer = NewStdoutWriter(log)
	}

	al := &AccessLogging{
		log:                  log.WithGroup("access-logging"),
		enabled:              cfg.Enabled,
		includeHeaders:       headers,
		includeOperationName: cfg.IncludeOperationName,
		includeVariables:     cfg.IncludeVariables,
		includePayload:       cfg.IncludePayload,
		async:                cfg.Async,
		writer:               writer,
		shutdown:             make(chan struct{}),
	}

	// Warn if async is enabled with GCP (it will be ignored)
	if cfg.Async && cfg.GoogleCloudLogging.Enabled {
		log.Warn("Async mode is not supported with Google Cloud Logging - ignoring async setting (GCP client handles batching internally)")
	}

	// Async buffering only for stdout (not used for GCP)
	if cfg.Async && cfg.Enabled && !cfg.GoogleCloudLogging.Enabled {
		al.logChan = make(chan logEntry, cfg.BufferSize)
		al.startAsyncLogger()
		bufferSizeGauge.Set(float64(cfg.BufferSize))
	} else {
		bufferSizeGauge.Set(0)
		bufferUsageGauge.Set(0)
	}

	return al, nil
}

func (a *AccessLogging) Log(payloads []gql.RequestData, headers http.Header) {
	if !a.enabled {
		return
	}

	if a.async {
		// Non-blocking send to channel
		select {
		case a.logChan <- logEntry{payloads: payloads, headers: headers}:
			// Successfully queued - update buffer usage metric
			bufferUsageGauge.Set(float64(len(a.logChan)))
		default:
			// Channel is full, drop the log entry to avoid blocking
			droppedLogsCounter.Inc()
		}
	} else {
		// Synchronous logging (original behavior)
		a.logSync(payloads, headers)
	}
}

func (a *AccessLogging) logSync(payloads []gql.RequestData, headers http.Header) {
	// Pre-filter headers once
	filteredHeaders := make(map[string]interface{}, len(a.includeHeaders))
	for key := range a.includeHeaders {
		filteredHeaders[key] = headers.Values(key)
	}

	data := LogEntryData{
		Payloads:        payloads,
		FilteredHeaders: filteredHeaders,
		IncludeOpName:   a.includeOperationName,
		IncludeVars:     a.includeVariables,
		IncludePayload:  a.includePayload,
	}

	a.writer.Write(data)
}

func (a *AccessLogging) startAsyncLogger() {
	a.wg.Go(func() {
		for {
			select {
			case entry := <-a.logChan:
				// Process each log entry immediately
				a.logSync(entry.payloads, entry.headers)
				// Update buffer usage metric after processing
				bufferUsageGauge.Set(float64(len(a.logChan)))

			case <-a.shutdown:
				// Drain remaining entries and exit
				for {
					select {
					case entry := <-a.logChan:
						a.logSync(entry.payloads, entry.headers)
						// Update buffer usage metric after processing
						bufferUsageGauge.Set(float64(len(a.logChan)))
					default:
						return
					}
				}
			}
		}
	})
}

// Shutdown gracefully stops the async logger and flushes the writer
func (a *AccessLogging) Shutdown(ctx context.Context) error {
	if !a.enabled {
		return nil
	}

	// Shutdown async logger if enabled
	if a.async {
		// Signal shutdown
		close(a.shutdown)

		// Wait for goroutine to finish with timeout
		done := make(chan struct{})
		go func() {
			a.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Continue to writer shutdown
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Shutdown the writer (e.g., flush GCP logs)
	return a.writer.Shutdown(ctx)
}
