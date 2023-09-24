package field_suggestions

import (
	"strings"
)

type Config struct {
	Enabled bool `conf:"default:true" yaml:"enabled"`
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

	if val, ok := payload["errors"]; ok {
		errors, ok := val.([]map[string]interface{})
		if ok {
			for _, err := range errors {
				if msg, ok := err["message"]; ok {
					if message, ok := msg.(string); ok {
						err["message"] = ReplaceSuggestions(message)
					}
				}
			}
		}
	}
	return payload
}

func ReplaceSuggestions(message string) string {
	if strings.HasPrefix(message, "Did you mean") {
		return "[redacted]"
	}
	return message
}
