package persistedoperations // nolint:revive

import (
	"cloud.google.com/go/storage"
	"context"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/api/iterator"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

var (
	reloadFilesGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   "graphql_protect",
		Subsystem:   "persisted_operations",
		Name:        "gcs_downloaded_files",
		Help:        "number of files downloaded from gcs",
		ConstLabels: nil,
	},
		[]string{})
)

// GcpStorageLoader loads persisted operations from a GCP Storage bucket.
// It matches files based on a `*.json` glob pattern and attempts to unmarshall them into
// a persisted operations map structure
type GcpStorageLoader struct {
	client *storage.Client
	bucket string
	store  string
	log    *slog.Logger
}

func init() {
	prometheus.MustRegister(reloadFilesGauge)
}

func NewGcpStorageLoader(ctx context.Context, bucket string, store string, logger *slog.Logger) (*GcpStorageLoader, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return &GcpStorageLoader{
		client: client,
		bucket: bucket,
		store:  store,
		log:    logger,
	}, nil
}
func (g *GcpStorageLoader) Load(ctx context.Context) error {
	it := g.client.Bucket(g.bucket).Objects(ctx, &storage.Query{
		MatchGlob:  "**.json",
		Versions:   false,
		Projection: storage.ProjectionNoACL,
	})

	numberOfFilesProcessed := 0

	var errs []error
	for {
		attrs, err := it.Next()

		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			// if any error is returned, any subsequent call returns the same error
			// so we break here
			errs = append(errs, err)
			break
		}

		err = g.processFile(ctx, attrs)
		if err != nil {
			errs = append(errs, err)
		}

		numberOfFilesProcessed++
	}

	g.log.Info("Read manifest files from bucket", "numFiles", numberOfFilesProcessed, "numErrs", len(errs))
	reloadFilesGauge.WithLabelValues().Set(float64(numberOfFilesProcessed))

	return errors.Join(errs...)
}

func (g *GcpStorageLoader) processFile(ctx context.Context, attrs *storage.ObjectAttrs) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	file, err := os.Create(filepath.Join(g.store, getFileName(attrs)))
	if err != nil {
		return fmt.Errorf("os.Create: %w", err)
	}
	defer file.Close()

	reader, err := g.client.Bucket(g.bucket).Object(attrs.Name).NewReader(ctx)
	if err != nil {
		cancel()
		return fmt.Errorf("Object(%q).NewReader: %w", attrs.Name, err)
	}
	defer reader.Close()

	written, err := io.Copy(file, reader)
	if err != nil {
		cancel()
		return fmt.Errorf("io.Copy: %w", err)
	}

	g.log.Info("Remote file downloaded", "file", file.Name(), "numBytes", written)

	if err = file.Sync(); err != nil {
		return fmt.Errorf("file.Sync: %w", err)
	}

	return nil
}

/*
GCS exposes files with their names including directories (e.g. 'folder/filename') since we only
care about the actual filename, we use this function to only get the filename e.g. /folder/filename becomes filename
*/
func getFileName(attrs *storage.ObjectAttrs) string {
	_, file := filepath.Split(attrs.Name)
	return file
}
