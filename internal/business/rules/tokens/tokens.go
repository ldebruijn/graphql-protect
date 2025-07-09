package tokens

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/lexer"
)

var resultCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "graphql_protect",
	Subsystem: "max_tokens",
	Name:      "results",
	Help:      "The results of the max tokens rule",
},
	[]string{"result"},
)

type Config struct {
	Enabled         bool           `yaml:"enabled"`
	Max             int            `yaml:"max"`
	RejectOnFailure bool           `yaml:"reject_on_failure"`
	Overrides       map[string]int `yaml:"overrides"`
}

func DefaultConfig() Config {
	return Config{
		Enabled:         true,
		Max:             1_000,
		RejectOnFailure: true,
		Overrides:       make(map[string]int),
	}
}

func init() {
	prometheus.MustRegister(resultCounter)
}

type MaxTokensRule struct {
	cfg Config
}

func MaxTokens(cfg Config) *MaxTokensRule {
	return &MaxTokensRule{
		cfg: cfg,
	}
}

func (t *MaxTokensRule) Validate(source *ast.Source, operationName string) error {
	if !t.cfg.Enabled {
		return nil
	}

	max := t.cfg.Max

	if value, ok := t.cfg.Overrides[operationName]; ok {
		max = value
	}

	lex := lexer.New(source)
	count := 0

	for {
		tok, err := lex.ReadToken()

		if err != nil {
			return err
		}

		if tok.Kind == lexer.EOF {
			break
		}

		count++
	}

	if count > max {
		if t.cfg.RejectOnFailure {
			resultCounter.WithLabelValues("rejected").Inc()
			return fmt.Errorf("operation has exceeded maximum tokens. found [%d], max [%d]", count, max)
		}
		resultCounter.WithLabelValues("failed").Inc()
	}
	resultCounter.WithLabelValues("allowed").Inc()
	return nil
}
