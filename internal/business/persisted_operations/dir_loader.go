package persisted_operations // nolint:revive

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
)

// DirLoader loads persisted operations from a filesystem directory
// It looks at all files in the directory, but doesn't traverse subdirectories
// If it finds a file with a `.json` extension it attempts to unmarshall it and use it as
// a source for persisted operations/
// If it fails to load a file it moves on to the next file in the directory
type DirLoader struct {
	path string
	log  *slog.Logger
}

func NewLocalDirLoader(cfg Config, log *slog.Logger) *DirLoader {
	return &DirLoader{
		path: cfg.Store,
		log:  log,
	}
}

func (d *DirLoader) Load(_ context.Context) (map[string]PersistedOperation, error) {
	files, err := os.ReadDir(d.path)
	if err != nil {
		// if we can't read the dir, try creating it
		err := os.Mkdir(d.path, 0750)
		if err != nil {
			return nil, err
		}
	}

	result := map[string]PersistedOperation{}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if filepath.Ext(file.Name()) == ".json" {
			filePath := filepath.Join(d.path, file.Name())
			contents, err := os.ReadFile(filePath)
			if err != nil {
				d.log.Warn("Error reading file", "err", err)
				continue
			}

			var manifestHashes map[string]string
			err = json.Unmarshal(contents, &manifestHashes)
			if err != nil {
				d.log.Warn("error unmarshalling operation file", "filepath", filePath, "err", err)
				continue
			}

			for hash, operation := range manifestHashes {
				result[hash] = NewPersistedOperation(operation)
			}
		}
	}

	return result, nil
}
