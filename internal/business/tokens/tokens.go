package tokens

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/lexer"
)

var resultCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "go_graphql_armor",
	Subsystem: "max_tokens",
	Name:      "results",
	Help:      "The results of the max tokens rule",
},
	[]string{"result"},
)

type Config struct {
	Enabled         bool `conf:"default:true" yaml:"enabled"`
	Max             int  `conf:"default:10000" yaml:"max"`
	RejectOnFailure bool `conf:"default:true" yaml:"reject-on-failure"`
}

type MaxTokensRule struct {
	cfg Config
}

func MaxTokens(cfg Config) *MaxTokensRule {
	return &MaxTokensRule{
		cfg: cfg,
	}
}

func (t *MaxTokensRule) Validate(source *ast.Source) error {
	if !t.cfg.Enabled {
		return nil
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

	if count > t.cfg.Max {
		if t.cfg.RejectOnFailure {
			resultCounter.WithLabelValues("rejected").Inc()
			return fmt.Errorf("operation has exceeded maximum tokens. found [%d], max [%d]", count, t.cfg.Max)
		}
		resultCounter.WithLabelValues("failed").Inc()
	}
	resultCounter.WithLabelValues("allowed").Inc()
	return nil
}
