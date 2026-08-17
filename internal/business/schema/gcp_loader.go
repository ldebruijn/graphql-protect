package schema

import (
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"
)

// objectReader abstracts reading a single GCS object, allowing test injection.
type objectReader interface {
	read(ctx context.Context) (string, error)
}

type gcsObjectReader struct {
	client *storage.Client
	bucket string
	object string
}

func (r *gcsObjectReader) read(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 50*time.Second)
	defer cancel()

	reader, err := r.client.Bucket(r.bucket).Object(r.object).NewReader(ctx)
	if err != nil {
		return "", fmt.Errorf("Object(%q).NewReader: %w", r.object, err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("io.ReadAll: %w", err)
	}

	return string(data), nil
}

type gcpSchemaLoader struct {
	reader objectReader
	bucket string
	object string
	log    *slog.Logger
}

func newGcpSchemaLoader(bucket, object string, log *slog.Logger) (*gcpSchemaLoader, error) {
	client, err := storage.NewClient(context.Background())
	if err != nil {
		return nil, err
	}

	return &gcpSchemaLoader{
		reader: &gcsObjectReader{
			client: client,
			bucket: bucket,
			object: object,
		},
		bucket: bucket,
		object: object,
		log:    log,
	}, nil
}

func (g *gcpSchemaLoader) fetch() (string, error) {
	contents, err := g.reader.read(context.Background())
	if err != nil {
		return "", err
	}
	g.log.Info("Loaded schema from GCS", "bucket", g.bucket, "object", g.object)
	return contents, nil
}
