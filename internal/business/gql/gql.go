package gql

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type RequestPayload struct {
	//OperationName string      `json:"operationName"`
	Variables  interface{} `json:"variables"`
	Query      string      `json:"query"`
	Extensions Extensions  `json:"extensions"`
}

type Extensions struct {
	PersistedQuery *PersistedQuery `json:"persistedQuery"`
}

type PersistedQuery struct {
	Sha256Hash string `json:"sha256Hash"`
}

func ParseRequestPayload(r *http.Request) (RequestPayload, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return RequestPayload{}, err
	}
	// Replace the body with a new reader after reading from the original
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	var payload RequestPayload
	err = json.Unmarshal(body, &payload)
	if err != nil {
		return RequestPayload{}, err
	}
	return payload, nil
}
