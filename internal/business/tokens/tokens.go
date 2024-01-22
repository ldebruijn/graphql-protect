package tokens

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/lexer"
	"time"
)

var resultHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "go_graphql_armor",
	Subsystem: "max_tokens",
	Name:      "results",
	Help:      "The results of the max tokens rule",
},
	[]string{"result"},
)

type Config struct {
	Enabled         bool `conf:"default:true" yaml:"enabled"`
	Max             int  `conf:"default:1000" yaml:"max"`
	RejectOnFailure bool `conf:"default:true" yaml:"reject_on_failure"`
}

func init() {
	prometheus.MustRegister(resultHistogram)
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

	start := time.Now()

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
			resultHistogram.WithLabelValues("rejected").Observe(time.Since(start).Seconds())
			return fmt.Errorf("operation has exceeded maximum tokens. found [%d], max [%d]", count, t.cfg.Max)
		}
		resultHistogram.WithLabelValues("failed").Observe(time.Since(start).Seconds())
	}
	resultHistogram.WithLabelValues("allowed").Observe(time.Since(start).Seconds())
	return nil
}
