package persisted_operations

import (
	"context"
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
	path                               string
	format                             string
	apolloPersistedQueryManifestParser ApolloPersistedQueryManifestParser
}

func NewLocalDirLoader(cfg Config) *DirLoader {
	return &DirLoader{
		path:                               cfg.Store,
		format:                             cfg.Format,
		apolloPersistedQueryManifestParser: ApolloPersistedQueryManifestParser{},
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

			if d.format == "apollo-persisted-query-manifest" {
				err = d.apolloPersistedQueryManifestParser.ParseContents(contents, result)
				if err != nil {
					continue
				}
			} else {
				fmt.Println("Not implemented format: ", d.format)
				continue
			}
		}
	}

	return result, nil
}
