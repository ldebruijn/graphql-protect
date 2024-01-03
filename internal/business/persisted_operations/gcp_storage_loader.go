package persisted_operations

import (
	"cloud.google.com/go/storage"
	"context"
	"errors"
	"fmt"
	"google.golang.org/api/iterator"
	"io"
	"os"
	"path/filepath"
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
func (gcp *GcpStorageLoader) Load(ctx context.Context) error {
	it := gcp.client.Bucket(gcp.bucket).Objects(ctx, &storage.Query{
		MatchGlob: "*.json",
	})

	var errs []error
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			break
		}

		ctx, cancel := context.WithTimeout(ctx, time.Second*50)

		file, err := os.Create(filepath.Join(gcp.store, attrs.Name))
		if err != nil {
			cancel()
			errs = append(errs, fmt.Errorf("os.Create: %w", err))
			continue
		}

		reader, err := gcp.client.Bucket(gcp.bucket).Object(attrs.Name).NewReader(ctx)
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

	return errors.Join(errs...)
}
