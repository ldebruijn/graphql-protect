package persisted_operations // nolint:revive

import (
	"cloud.google.com/go/storage"
	"context"
	"errors"
	"fmt"
	"google.golang.org/api/iterator"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GcpStorageLoader loads persisted operations from a GCP Storage bucket.
// It matches files based on a `*.json` glob pattern and attempts to unmarshall them into
// a persisted operations map structure
type GcpStorageLoader struct {
	client *storage.Client
	bucket string
	store  string
}

func NewGcpStorageLoader(ctx context.Context, bucket string, store string) (*GcpStorageLoader, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return &GcpStorageLoader{
		client: client,
		bucket: bucket,
		store:  store,
	}, nil
}
func (g *GcpStorageLoader) Load(ctx context.Context, log *slog.Logger) error {
	it := g.client.Bucket(g.bucket).Objects(ctx, &storage.Query{
		MatchGlob:  "**.json",
		Versions:   false,
		Projection: storage.Projection(2), //ProjectionNoACL to speed up downloading
	})

	var numberOfFilesProcessed = 0

	var errs []error
	for {
		attrs, err := it.Next()
		numberOfFilesProcessed++

		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			errs = append(errs, err)
			break
		}

		ctx, cancel := context.WithTimeout(ctx, time.Second*50)

		file, err := os.Create(filepath.Join(g.store, getFileName(attrs)))
		if err != nil {
			cancel()
			errs = append(errs, fmt.Errorf("os.Create: %w", err))
			continue
		}

		reader, err := g.client.Bucket(g.bucket).Object(attrs.Name).NewReader(ctx)
		if err != nil {
			cancel()
			errs = append(errs, fmt.Errorf("Object(%q).NewReader: %w", attrs.Name, err))
			continue
		}

		if _, err := io.Copy(file, reader); err != nil {
			cancel()
			errs = append(errs, fmt.Errorf("io.Copy: %w", err))
			continue
		}

		if err = file.Close(); err != nil {
			cancel()
			errs = append(errs, fmt.Errorf("file.Close: %w", err))
		}

		cancel()
		_ = reader.Close()
	}

	log.Info(fmt.Sprintf("Read %d manifest files from bucket", numberOfFilesProcessed))

	return errors.Join(errs...)
}

func getFileName(attrs *storage.ObjectAttrs) string {
	var fileNameSplit = strings.Split(attrs.Name, "/")
	var fileName = fileNameSplit[len(fileNameSplit)-1]
	return fileName
}
