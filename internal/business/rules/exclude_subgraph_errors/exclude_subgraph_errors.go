package exclude_subgraph_errors // nolint:revive

type Config struct {
	Enabled bool `conf:"default:true" yaml:"enabled"`
}

type ExcludeSubgraphErrors struct {
	enabled bool
}

func NewExcludeSubgraphErrors(cfg Config) *ExcludeSubgraphErrors {
	return &ExcludeSubgraphErrors{
		enabled: cfg.Enabled,
	}
}

func (a *ExcludeSubgraphErrors) ProcessBody(payload map[string]interface{}) map[string]interface{} {
	redactedErrorArray := []map[string]interface{}{
		{
			"message": "Subgraph errors redacted",
		},
	}

	if payload["errors"] != nil {
		payload["errors"] = redactedErrorArray
	}

	return payload
}

func (a *ExcludeSubgraphErrors) Enabled() bool {
	return a.enabled
}

type GraphqlError struct {
	Message    string                 `json:"message"`
	Path       []string               `json:"path"`
	Extensions map[string]interface{} `json:"extensions"`
}
