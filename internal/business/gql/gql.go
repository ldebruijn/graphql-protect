package gql

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type RequestData struct {
	Variables  map[string]interface{} `json:"variables,omitempty"`
	Query      string                 `json:"query,omitempty"`
	Extensions Extensions             `json:"extensions,omitempty"`
}

type Extensions struct {
	PersistedQuery *PersistedQuery `json:"persistedQuery,omitempty"`
}

type PersistedQuery struct {
	Sha256Hash string `json:"sha256Hash"`
}

func ParseRequestPayload(r *http.Request) ([]RequestData, error) {
	if r.ContentLength < 1 {
		return []RequestData{}, nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return []RequestData{}, err
	}
	// Replace the body with a new reader after reading from the original
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	body = bytes.TrimSpace(body)
	// assume its a batch request
	if body[0] == '[' {
		var data []RequestData
		err = json.Unmarshal(body, &data)
		if err != nil {
			return []RequestData{}, err
		}
		return data, nil
	}
	var data RequestData
	err = json.Unmarshal(body, &data)
	if err != nil {
		return []RequestData{}, err
	}
	return []RequestData{data}, nil
}
