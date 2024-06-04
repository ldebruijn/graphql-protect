package accesslogging

import (
	"github.com/ldebruijn/graphql-protect/internal/business/gql"
	"log/slog"
	"net/http"
)

type Config struct {
	Enabled              bool     `config:"default:true" yaml:"enabled"`
	IncludedHeaders      []string `yaml:"included_headers"`
	IncludeOperationName bool     `config:"default:true" yaml:"include_operation_name"`
	IncludeVariables     bool     `config:"default:true" yaml:"include_variables"`
	IncludePayload       bool     `config:"default:false" yaml:"include_payload"`
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

	toLog := map[string]interface{}{}

	logHeaders := map[string]interface{}{}
	for key, _ := range a.includeHeaders {
		logHeaders[key] = headers.Values(key)
	}

	for _, req := range payloads {
		if a.includeOperationName {
			toLog["operationName"] = req.OperationName
		}
		if a.includeVariables {
			toLog["variables"] = req.Variables
		}
		if a.includePayload {
			toLog["payload"] = req.Query
		}

		toLog["headers"] = logHeaders

		a.log.Info("record", "payload", toLog)
	}
}
