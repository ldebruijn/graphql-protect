package accesslogging

import (
	"context"
	"log/slog"
	"net/http"
	"sync"

	"github.com/ldebruijn/graphql-protect/internal/business/gql"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	droppedLogsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "graphql_protect",
		Subsystem: "access_logging",
		Name:      "dropped_logs_total",
		Help:      "The total number of access log entries dropped due to full channel buffer",
	},
		[]string{"reason"},
	)
)

func init() {
	prometheus.MustRegister(droppedLogsCounter)
}

type Config struct {
	Enabled              bool     `yaml:"enabled"`
	IncludedHeaders      []string `yaml:"include_headers"`
	IncludeOperationName bool     `yaml:"include_operation_name"`
	IncludeVariables     bool     `yaml:"include_variables"`
	IncludePayload       bool     `yaml:"include_payload"`
	Async                bool     `yaml:"async"`
	BufferSize           int      `yaml:"buffer_size"`
}

func DefaultConfig() Config {
	return Config{
		Enabled:              true,
		IncludedHeaders:      nil,
		IncludeOperationName: true,
		IncludeVariables:     true,
		IncludePayload:       false,
		Async:                false,
		BufferSize:           1000,
	}
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
	logChan              chan logEntry
	shutdown             chan struct{}
	wg                   sync.WaitGroup
}

func NewAccessLogging(cfg Config, log *slog.Logger) *AccessLogging {
	headers := map[string]bool{}
	for _, header := range cfg.IncludedHeaders {
		headers[header] = true
	}

	al := &AccessLogging{
		log:                  log.WithGroup("access-logging"),
		enabled:              cfg.Enabled,
		includeHeaders:       headers,
		includeOperationName: cfg.IncludeOperationName,
		includeVariables:     cfg.IncludeVariables,
		includePayload:       cfg.IncludePayload,
		async:                cfg.Async,
		shutdown:             make(chan struct{}),
	}

	if cfg.Async && cfg.Enabled {
		al.logChan = make(chan logEntry, cfg.BufferSize)
		al.startAsyncLogger()
	}

	return al
}

func (a *AccessLogging) Log(payloads []gql.RequestData, headers http.Header) {
	if !a.enabled {
		return
	}

	if a.async {
		// Non-blocking send to channel
		select {
		case a.logChan <- logEntry{payloads: payloads, headers: headers}:
			// Successfully queued
		default:
			// Channel is full, drop the log entry to avoid blocking
			droppedLogsCounter.WithLabelValues("channel_full").Inc()
		}
	} else {
		// Synchronous logging (original behavior)
		a.logSync(payloads, headers)
	}
}

func (a *AccessLogging) logSync(payloads []gql.RequestData, headers http.Header) {
	headersToInclude := map[string]interface{}{}
	for key := range a.includeHeaders {
		headersToInclude[key] = headers.Values(key)
	}

	for _, req := range payloads {
		al := accessLog{}

		if a.includeOperationName {
			al.WithOperationName(req.OperationName)
		}
		if a.includeVariables {
			al.WithVariables(req.Variables)
		}
		if a.includePayload {
			al.WithPayload(req.Query)
		}

		al.WithHeaders(headersToInclude)

		a.log.Info("record", "payload", al)
	}
}

func (a *AccessLogging) startAsyncLogger() {
	a.wg.Go(func() {
		for {
			select {
			case entry := <-a.logChan:
				// Process each log entry immediately
				a.logSync(entry.payloads, entry.headers)

			case <-a.shutdown:
				// Drain remaining entries and exit
				for {
					select {
					case entry := <-a.logChan:
						a.logSync(entry.payloads, entry.headers)
					default:
						return
					}
				}
			}
		}
	})
}

// Shutdown gracefully stops the async logger
func (a *AccessLogging) Shutdown(ctx context.Context) error {
	if !a.async || !a.enabled {
		return nil
	}

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
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
