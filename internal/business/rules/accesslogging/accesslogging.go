package accesslogging

import (
	"github.com/ldebruijn/graphql-protect/internal/business/gql"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"log/slog"
	"net/http"
	"slices"
)

type Config struct {
	Enable               bool     `config:"default:true" yaml:"enabled"`
	IncludedHeaders      []string `yaml:"included_headers"`
	IncludeOperationName bool     `config:"default:true" yaml:"include_operation_name"`
	IncludeVariables     bool     `config:"default:true" yaml:"include_variables"`
	IncludePayload       bool     `config:"default:false" yaml:"include_payload"`
}

type AccessLogging struct {
	log *slog.Logger
	cfg Config
}

func NewAccessLogging(cfg Config) *AccessLogging {
	return &AccessLogging{
		// otelslog bridge for pushing otellogs directly to connector
		log: otelslog.NewLogger("access-logging"),
		cfg: cfg,
	}
}

func (a *AccessLogging) Log(payloads []gql.RequestData, headers http.Header) {
	if !a.cfg.Enable {
		return
	}

	toLog := map[string]interface{}{}

	for _, req := range payloads {
		if a.cfg.IncludeOperationName {
			toLog["operationName"] = req.OperationName
		}
		if a.cfg.IncludeVariables {
			toLog["variables"] = req.Variables
		}
		if a.cfg.IncludePayload {
			toLog["payload"] = req.Query
		}

		logHeaders := map[string]interface{}{}
		for name, values := range headers {
			if slices.Contains(a.cfg.IncludedHeaders, name) {
				logHeaders[name] = values
			}
		}

		toLog["headers"] = logHeaders

		a.log.Info("Access Logging", "payload", toLog)
	}
}
