package persisted_operations

import (
	"cloud.google.com/go/storage"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/api/iterator"
	"io"
)

// GcpStorageLoader loads persisted operations from a GCP Storage bucket.
// It matches files based on a `*.json` glob pattern and attempts to unmarshall them into
// a persisted operations map structure
type GcpStorageLoader struct {
	client *storage.Client
	bucket string
}

func NewGcpStorageLoader(ctx context.Context, bucket string) (*GcpStorageLoader, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return &GcpStorageLoader{
		client: client,
		bucket: bucket,
	}, nil
}

func (g *GcpStorageLoader) Load(ctx context.Context) (map[string]string, error) {
	var store map[string]string

	it := g.client.Bucket(g.bucket).Objects(ctx, &storage.Query{
		MatchGlob: "*.json",
	})
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			continue
		}

		rc, err := g.client.Bucket(g.bucket).Object(attrs.Name).NewReader(ctx)
		if err != nil {
			continue
		}

		data, err := io.ReadAll(rc)
		if err != nil {
			return nil, fmt.Errorf("ioutil.ReadAll: %w", err)
		}
		_ = rc.Close()

		err = json.Unmarshal(data, &store)
		if err != nil {
			continue
		}
	}

	return store, nil
}
