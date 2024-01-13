package block_field_suggestions // nolint:revive

import (
	"github.com/prometheus/client_golang/prometheus"
	"strings"
)

var resultCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "go_graphql_armor",
	Subsystem: "block_field_suggestions",
	Name:      "results",
	Help:      "The results of the block field suggestions rule",
},
	[]string{"result"},
)

type Config struct {
	Enabled bool   `conf:"default:true" yaml:"enabled"`
	Mask    string `conf:"default:[redacted]" yaml:"redacted"`
}

type BlockFieldSuggestionsHandler struct {
	cfg Config
}

func init() {
	prometheus.MustRegister(resultCounter)
}

func NewBlockFieldSuggestionsHandler(cfg Config) *BlockFieldSuggestionsHandler {
	return &BlockFieldSuggestionsHandler{
		cfg: cfg,
	}
}

func (b *BlockFieldSuggestionsHandler) Enabled() bool {
	return b.cfg.Enabled
}

func (b *BlockFieldSuggestionsHandler) ProcessBody(payload map[string]interface{}) map[string]interface{} {
	if val, ok1 := payload["errors"]; ok1 {
		payload["errors"] = b.processErrors(val)
	}
	return payload
}

func (b *BlockFieldSuggestionsHandler) processErrors(payload interface{}) interface{} {
	switch payload := payload.(type) {
	case []map[string]interface{}:
		for _, err := range payload {
			_ = b.processError(err)
		}
	case []interface{}:
		for _, err := range payload {
			e, ok2 := err.(map[string]interface{})
			if !ok2 {
				continue
			}
			_ = b.processError(e)
		}
	}
	return payload
}

func (b *BlockFieldSuggestionsHandler) processError(err map[string]interface{}) map[string]interface{} {
	if msg, ok4 := err["message"]; ok4 {
		if message, ok := msg.(string); ok {
			err["message"] = b.replaceSuggestions(message)
		}
	}
	return err
}

func (b *BlockFieldSuggestionsHandler) replaceSuggestions(message string) string {
	if strings.HasPrefix(message, "Did you mean") {
		resultCounter.WithLabelValues("masked").Inc()
		return b.cfg.Mask
	}
	resultCounter.WithLabelValues("unmasked").Inc()
	return message
}
