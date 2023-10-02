package persisted_operations

import (
	"context"
	"encoding/json"
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
}

func NewLocalDirLoader(cfg Config) *DirLoader {
	return &DirLoader{
		path: cfg.Store,
	}
}

func (d *DirLoader) Load(ctx context.Context) (map[string]string, error) {
	files, err := os.ReadDir(d.path)
	if err != nil {
		return nil, err
	}

	var result map[string]string

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if filepath.Ext(file.Name()) == ".json" {
			contents, err := os.ReadFile(file.Name())
			if err != nil {
				continue
			}
			// append to map
			err = json.Unmarshal(contents, &result)
			if err != nil {
				continue
			}
		}
	}

	return result, nil
}
