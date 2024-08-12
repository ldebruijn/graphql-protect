package persistedoperations

import (
	"encoding/json"
	"fmt"
	"strings"
)

type PersistedOperation struct {
	Operation string
	Name      string
}

func UnmarshallPersistedOperations(payload []byte) (map[string]PersistedOperation, error) {
	var manifestHashes map[string]string
	err := json.Unmarshal(payload, &manifestHashes)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling operation file, bytes: %d, contents: %s, error: %w", len(payload), string(payload), err)
	}

	data := make(map[string]PersistedOperation)

	for hash, operation := range manifestHashes {
		data[hash] = NewPersistedOperation(operation)
	}
	return data, nil
}

func NewPersistedOperation(operation string) PersistedOperation {
	name := extractOperationNameFromPersistedOperation(operation)
	return PersistedOperation{
		Operation: operation,
		Name:      name,
	}
}

func extractOperationNameFromPersistedOperation(payload string) string {
	firstSpace := strings.Index(payload, " ")
	firstBracket := strings.Index(payload, "{")
	firstParenthesis := strings.Index(payload, "(")

	until := firstBracket
	if firstParenthesis < firstBracket {
		until = firstParenthesis
	}

	if firstSpace > until || until == -1 {
		return ""
	}

	return strings.TrimSpace(payload[firstSpace+1 : until])
}
