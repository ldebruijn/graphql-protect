package trusteddocuments

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// find the first word after the 'query' or 'mutation' keyword
var operationNameRegexPattern = regexp.MustCompile(`\b(query|mutation)\s(\w+)`)

type PersistedOperation struct {
	Operation string
	Name      string `json:"name,omitempty"`
}

func unmarshallPersistedOperations(payload []byte) (map[string]PersistedOperation, error) {
	var manifestHashes map[string]string

	err := json.Unmarshal(payload, &manifestHashes)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling operation file, bytes: %d, contents: %s, error: %w", len(payload), string(payload), err)
	}

	data := make(map[string]PersistedOperation)

	for hash, operation := range manifestHashes {
		data[hash] = PersistedOperation{
			Operation: operation,
			Name:      extractOperationNameFromOperation(operation),
		}
	}
	return data, nil
}

func extractOperationNameFromOperation(payload string) string {
	match := operationNameRegexPattern.FindStringSubmatch(payload)

	// match[0] is the entire match
	// match[1] is either mutation/query
	// match[2] is the name of the operation
	if len(match) == 3 {
		return match[2]
	}
	return ""
}
