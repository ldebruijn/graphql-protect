package persisted_operations

import (
	"context"
	"encoding/json"
	"fmt"
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
		// if we can't read the dir, try creating it
		err := os.Mkdir(d.path, 0750)
		if err != nil {
			return nil, err
		}
	}

	var result = make(map[string]string)

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if filepath.Ext(file.Name()) == ".json" {
			filePath := fmt.Sprintf("%s/%s", d.path, file.Name())
			contents, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}

			err = json.Unmarshal(contents, &result)

			if err != nil {
				fmt.Println(err)
				continue
			}
		}
	}

	return result, nil
}
