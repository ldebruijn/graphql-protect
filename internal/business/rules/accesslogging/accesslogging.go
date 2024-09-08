package accesslogging

import (
	"github.com/ldebruijn/graphql-protect/internal/business/gql"
	"log/slog"
	"net/http"
)

type Config struct {
	Enabled              bool     `yaml:"enabled"`
	IncludedHeaders      []string `yaml:"include_headers"`
	IncludeOperationName bool     `yaml:"include_operation_name"`
	IncludeVariables     bool     `yaml:"include_variables"`
	IncludePayload       bool     `yaml:"include_payload"`
}

func DefaultConfig() Config {
	return Config{
		Enabled:              true,
		IncludedHeaders:      nil,
		IncludeOperationName: true,
		IncludeVariables:     true,
		IncludePayload:       false,
	}
}

type AccessLogging struct {
	log                  *slog.Logger
	enabled              bool
	includeHeaders       map[string]bool
	includeOperationName bool
	includeVariables     bool
	includePayload       bool
}

func NewAccessLogging(cfg Config, log *slog.Logger) *AccessLogging {
	headers := map[string]bool{}
	for _, header := range cfg.IncludedHeaders {
		headers[header] = true
	}

	return &AccessLogging{
		log:                  log.WithGroup("access-logging"),
		enabled:              cfg.Enabled,
		includeHeaders:       headers,
		includeOperationName: cfg.IncludeOperationName,
		includeVariables:     cfg.IncludeVariables,
		includePayload:       cfg.IncludePayload,
	}
}

func (a *AccessLogging) Log(payloads []gql.RequestData, headers http.Header) {
	if !a.enabled {
		return
	}

	headersToInclude := map[string]interface{}{}
	for key := range a.includeHeaders {
		headersToInclude[key] = headers.Values(key)
	}

	for _, req := range payloads {
		al := accessLog{}

		if a.includeOperationName {
			al.WithOperationName(req.OperationName)
		}
		if a.includeVariables {
			al.WithVariables(req.Variables)
		}
		if a.includePayload {
			al.WithPayload(req.Query)
		}

		al.WithHeaders(headersToInclude)

		a.log.Info("record", "payload", al)
	}
}
