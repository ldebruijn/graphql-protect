package persistedoperations // nolint:revive

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
)

var _ Loader = &LocalLoader{}

var (
	fileLoaderGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   "graphql_protect",
		Subsystem:   "dir_loader",
		Name:        "files_loaded_gauge",
		Help:        "number of files loaded from disk",
		ConstLabels: nil,
	}, []string{})
)

// LocalLoader loads persisted operations from a filesystem directory
// It looks at all files in the directory, but doesn't traverse subdirectories
// If it finds a file with a `.json` extension it attempts to unmarshall it and use it as
// a source for persisted operations/
// If it fails to load a file it moves on to the next file in the directory
type LocalLoader struct {
	cfg Config
	log *slog.Logger
}

func (d *LocalLoader) Type() string {
	return "local"
}

func NewLocalDirLoader(cfg Config, log *slog.Logger) *LocalLoader {
	return &LocalLoader{
		cfg: cfg,
		log: log,
	}
}

func init() {
	prometheus.MustRegister(fileLoaderGauge)
}

func (d *LocalLoader) Load(_ context.Context) (map[string]PersistedOperation, error) {
	files, err := os.ReadDir(d.cfg.Loader.Location)
	if err != nil {
		// if we can't read the dir, try creating it
		err := os.Mkdir(d.cfg.Loader.Location, 0750)
		if err != nil {
			return nil, err
		}
	}

	result := map[string]PersistedOperation{}
	var filesProcessed = 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if filepath.Ext(file.Name()) == ".json" {
			filePath := filepath.Join(d.cfg.Loader.Location, file.Name())
			contents, err := os.ReadFile(filePath)
			if err != nil {
				d.log.Warn("Error reading file", "err", err)
				continue
			}

			filesProcessed++

			data, err := UnmarshallPersistedOperations(contents)
			if err != nil {
				d.log.Warn("error unmarshalling operation file", "bytes", len(contents), "contents", string(contents), "filepath", filePath, "err", err)
				continue
			}

			maps.Copy(result, data)
		}
		return result, err
	}

	fileLoaderGauge.WithLabelValues().Set(float64(filesProcessed))

	return result, nil
}
