package persisted_operations

import (
	"encoding/json"
)

type ApolloPersistedQueryManifestParser struct {
}

func (a *ApolloPersistedQueryManifestParser) ParseContents(contents []byte, result map[string]string) error {

	var persistedQueryManifest PersistedQueryManifest
	err := json.Unmarshal(contents, &persistedQueryManifest)

	for _, value := range persistedQueryManifest.Operations {
		result[value.Hash] = value.Query
	}

	return err
}

type Operation struct {
	Hash  string `json:"id"`
	Query string `json:"body"`
	Name  string `json:"name"`
	Type  string `json:"type"`
}

type PersistedQueryManifest struct {
	Format     string      `json:"format"`
	Version    int         `json:"version"`
	Operations []Operation `json:"operations"`
}
