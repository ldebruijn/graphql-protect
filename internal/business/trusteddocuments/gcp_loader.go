package trusteddocuments // nolint:revive

import (
	"cloud.google.com/go/storage"
	"context"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/api/iterator"
	"io"
	"log/slog"
	"maps"
	"time"
)

var _ Loader = &GcpLoader{}

var (
	filesLoadedCounter = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   "graphql_protect",
		Subsystem:   "persisted_operations",
		Name:        "gcs_loader_files_loaded_count",
		Help:        "number of files downloaded using gcs loader",
		ConstLabels: nil,
	},
		[]string{})
)

// GcpStorageLoader loads persisted operations from a GCP Storage bucket.
// It matches files based on a `*.json` glob pattern and attempts to unmarshall them into
// a persisted operations map structure
type GcpLoader struct {
	client *storage.Client
	bucket string
	log    *slog.Logger
}

func (g *GcpLoader) Type() string {
	return "gcp"
}

func init() {
	prometheus.MustRegister(filesLoadedCounter)
}

func NewGcpLoader(cfg LoaderConfig, log *slog.Logger) (*GcpLoader, error) {
	client, err := storage.NewClient(context.Background())
	if err != nil {
		return nil, err
	}

	return &GcpLoader{
		client: client,
		bucket: cfg.Location,
		log:    log,
	}, nil
}
func (g *GcpLoader) Load(ctx context.Context) (map[string]PersistedOperation, error) {
	it := g.client.Bucket(g.bucket).Objects(ctx, &storage.Query{
		MatchGlob:  "**.json",
		Versions:   false,
		Projection: storage.ProjectionNoACL,
	})

	numberOfFilesProcessed := 0

	result := map[string]PersistedOperation{}
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

		data, err := g.processFile(ctx, attrs)
		if err != nil {
			errs = append(errs, err)
		}

		maps.Copy(result, data)

		numberOfFilesProcessed++
	}

	g.log.Info("Loaded files from gcp bucket", "numFiles", numberOfFilesProcessed, "numErrs", len(errs))
	filesLoadedCounter.WithLabelValues().Set(float64(numberOfFilesProcessed))

	return result, errors.Join(errs...)
}

func (g *GcpLoader) processFile(ctx context.Context, attrs *storage.ObjectAttrs) (map[string]PersistedOperation, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	reader, err := g.client.Bucket(g.bucket).Object(attrs.Name).NewReader(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("Object(%q).NewReader: %w", attrs.Name, err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("io.Copy: %w", err)
	}

	operations, err := unmarshallPersistedOperations(data)

	for _, operation := range operations {
		if operation.Name == "" {
			g.log.Warn("Operation without operation name found!", "operation", operation)
		}
	}

	return operations, err
}
