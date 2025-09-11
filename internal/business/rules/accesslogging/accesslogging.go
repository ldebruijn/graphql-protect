package accesslogging

import (
	"bufio"
	"context"
	"log/slog"
	"net/http"
	"os"
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
	stdoutWriter         *bufio.Writer
}

func NewAccessLogging(cfg Config, log *slog.Logger) *AccessLogging {
	headers := map[string]bool{}
	for _, header := range cfg.IncludedHeaders {
		headers[header] = true
	}

	// construct a buffered writer on stdout to prevent backpressure on stdout from heavy access logs volumes
	stdoutWriter := bufio.NewWriterSize(os.Stdout, cfg.BufferSize)
	if !cfg.Async {
		// use buffered stdout writer for access logs, always use json format for access logs
		log = slog.New(slog.NewJSONHandler(stdoutWriter, nil)).WithGroup("access-logging")
	}

	al := &AccessLogging{
		log:                  log,
		enabled:              cfg.Enabled,
		includeHeaders:       headers,
		includeOperationName: cfg.IncludeOperationName,
		includeVariables:     cfg.IncludeVariables,
		includePayload:       cfg.IncludePayload,
		async:                cfg.Async,
		shutdown:             make(chan struct{}),
		stdoutWriter:         stdoutWriter,
	}

	if cfg.Async && cfg.Enabled {
		al.logChan = make(chan logEntry, cfg.BufferSize)
		al.startAsyncLogger()
		// Set the buffer size limit metric
		bufferSizeGauge.Set(float64(cfg.BufferSize))
	} else {
		// Reset buffer metrics when not using async logging
		bufferSizeGauge.Set(0)
		bufferUsageGauge.Set(0)
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

// Shutdown gracefully stops the async logger
func (a *AccessLogging) Shutdown(ctx context.Context) error {
	if !a.enabled {
		return nil
	}

	if !a.async {
		// ensure last logs are flushed upon shutdown
		return a.stdoutWriter.Flush()
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
