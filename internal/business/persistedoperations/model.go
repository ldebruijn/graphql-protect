package persistedoperations

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
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
	var errs []error

	for hash, operation := range manifestHashes {
		data[hash], err = NewPersistedOperation(operation)
		errs = append(errs, err)
	}
	return data, errors.Join(errs...)
}

func NewPersistedOperation(operation string) (PersistedOperation, error) {
	name, err := extractOperationNameFromPersistedOperation(operation)

	if err == nil {
		return PersistedOperation{
			Operation: operation,
			Name:      name,
		}, nil
	} else { //Don't fill in the operationName if we cant parse it

		return PersistedOperation{
			Operation: operation,
		}, err
	}
}

func extractOperationNameFromPersistedOperation(payload string) (string, error) {
	// find the first word after the 'query' or 'mutation' keyword
	pattern := `\b(query|mutation)\s(\w+)`

	re := regexp.MustCompile(pattern)

	match := re.FindStringSubmatch(payload)

	// match[0] is the entire match
	// match[1] is either mutation/query
	// match[2] is the name of the operation
	if len(match) == 3 {
		return match[2], nil
	} else {
		return "", fmt.Errorf("no operation name match found for query/mutation %s", payload)
	}
}
