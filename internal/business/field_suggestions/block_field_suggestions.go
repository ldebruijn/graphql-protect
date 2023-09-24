package field_suggestions

import (
	"strings"
)

type Config struct {
	Enabled bool `conf:"default:true" yaml:"enabled"`
	//Mask    string `conf:"default:[redacted]" yaml:"redacted"`
}

type FieldSuggestionsHandler struct {
	cfg Config
}

func NewFieldSuggestionsHandler(cfg Config) *FieldSuggestionsHandler {
	return &FieldSuggestionsHandler{
		cfg: cfg,
	}
}

func ProcessBody(payload map[string]interface{}) map[string]interface{} {
	if val, ok1 := payload["errors"]; ok1 {
		payload["errors"] = processErrors(val)
	}
	return payload
}

func processErrors(payload interface{}) interface{} {
	switch payload.(type) {
	case []map[string]interface{}:
		for _, err := range payload.([]map[string]interface{}) {
			err = processError(err)
		}
	case []interface{}:
		for _, err := range payload.([]interface{}) {
			e, ok2 := err.(map[string]interface{})
			if !ok2 {
				continue
			}
			e = processError(e)
		}
	}
	return payload
}

func processError(err map[string]interface{}) map[string]interface{} {
	if msg, ok4 := err["message"]; ok4 {
		if message, ok := msg.(string); ok {
			err["message"] = ReplaceSuggestions(message)
		}
	}
	return err
}

func ReplaceSuggestions(message string) string {
	if strings.HasPrefix(message, "Did you mean") {
		return "[redacted]"
	}
	return message
}
