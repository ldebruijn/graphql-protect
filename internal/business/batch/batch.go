package batch

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/lexer"
)

var resultCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "go_graphql_armor",
	Subsystem: "max_batch",
	Name:      "results",
	Help:      "The results of the max batch rule",
},
	[]string{"result"},
)

type Config struct {
	Enabled         bool `conf:"default:true" yaml:"enabled"`
	Max             int  `conf:"default:3" yaml:"max"`
	RejectOnFailure bool `conf:"default:true" yaml:"reject_on_failure"`
}

func init() {
	prometheus.MustRegister(resultCounter)
}

type MaxBatchRule struct {
	cfg Config
}

func MaxBatch(cfg Config) *MaxBatchRule {
	return &MaxBatchRule{
		cfg: cfg,
	}
}

func (t *MaxBatchRule) Validate(source *ast.Source) error {
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
