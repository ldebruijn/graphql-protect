package accesslogging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"cloud.google.com/go/logging"
	"github.com/prometheus/client_golang/prometheus"
)

var _ LogWriter = &GoogleCloudWriter{}

var (
	gcpWritesCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "graphql_protect",
		Subsystem: "access_logging",
		Name:      "gcp_writes_total",
		Help:      "The total number of access log entries written to Google Cloud Logging",
	})

	gcpErrorsCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "graphql_protect",
		Subsystem: "access_logging",
		Name:      "gcp_errors_total",
		Help:      "The total number of errors encountered while writing to Google Cloud Logging",
	})
)

func init() {
	prometheus.MustRegister(gcpWritesCounter, gcpErrorsCounter)
}

// GoogleCloudWriter writes access logs to Google Cloud Logging
type GoogleCloudWriter struct {
	client *logging.Client
	logger *logging.Logger
	log    *slog.Logger
}

// NewGoogleCloudWriter creates a new Google Cloud Logging writer
func NewGoogleCloudWriter(cfg GoogleCloudConfig, log *slog.Logger) (*GoogleCloudWriter, error) {
	projectID, err := detectProjectID(cfg.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to detect GCP project ID: %w", err)
	}

	ctx := context.Background()
	client, err := logging.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create Google Cloud Logging client: %w", err)
	}

	logName := cfg.LogName
	if logName == "" {
		logName = "graphql-protect-access-logs"
	}

	logger := client.Logger(logName)

	log.Info("Google Cloud Logging initialized",
		"project_id", projectID,
		"log_name", logName,
	)

	return &GoogleCloudWriter{
		client: client,
		logger: logger,
		log:    log,
	}, nil
}

// Write writes a log entry to Google Cloud Logging
func (w *GoogleCloudWriter) Write(data LogEntryData) {
	logs := buildAccessLogs(data)

	for _, al := range logs {
		// The Log() method is non-blocking and batches entries automatically
		w.logger.Log(logging.Entry{
			Severity: logging.Info,
			Payload:  al,
		})
	}

	// Batch increment metrics
	gcpWritesCounter.Add(float64(len(logs)))
}

// Shutdown flushes pending logs and closes the client
func (w *GoogleCloudWriter) Shutdown(_ context.Context) error {
	w.log.Info("Shutting down Google Cloud Logging writer")

	var errs []error

	// Flush any buffered log entries
	if err := w.logger.Flush(); err != nil {
		gcpErrorsCounter.Inc()
		errs = append(errs, fmt.Errorf("flush failed: %w", err))
	}

	// Always close the client, even if flush failed
	if err := w.client.Close(); err != nil {
		gcpErrorsCounter.Inc()
		errs = append(errs, fmt.Errorf("close failed: %w", err))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	w.log.Info("Google Cloud Logging writer shutdown complete")
	return nil
}

// detectProjectID requires the GCP project ID from config
func detectProjectID(configProjectID string) (string, error) {
	if configProjectID == "" {
		return "", fmt.Errorf("GCP project ID not configured. Set project_id in the google_cloud_logging configuration")
	}
	return configProjectID, nil
}
