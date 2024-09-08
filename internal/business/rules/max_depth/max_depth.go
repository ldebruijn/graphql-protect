package max_depth // nolint:revive

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/validator"
)

var resultCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "graphql_protect",
	Subsystem: "max_depth",
	Name:      "results",
	Help:      "The results of the max_depth rule",
},
	[]string{"result"},
)

type Config struct {
	Enabled         bool `yaml:"enabled"`
	Max             int  `yaml:"max"`
	RejectOnFailure bool `yaml:"reject_on_failure"`
}

func DefaultConfig() Config {
	return Config{
		Enabled:         true,
		Max:             15,
		RejectOnFailure: true,
	}
}

func init() {
	prometheus.MustRegister(resultCounter)
}

func NewMaxDepthRule(cfg Config) {
	if cfg.Enabled {
		validator.AddRule("MaxDepth", func(observers *validator.Events, addError validator.AddErrFunc) {
			observers.OnOperation(func(_ *validator.Walker, operation *ast.OperationDefinition) {
				var maxDepth = countDepth(operation.SelectionSet)

				if maxDepth > cfg.Max {
					if cfg.RejectOnFailure {
						addError(
							validator.Message("syntax error: Depth limit of %d exceeded, found %d", cfg.Max, maxDepth),
							validator.At(operation.Position),
						)
						resultCounter.WithLabelValues("rejected").Inc()
					} else {
						resultCounter.WithLabelValues("failed").Inc()
					}
				} else {
					resultCounter.WithLabelValues("allowed").Inc()
				}
			})
		})
	}
}

func countDepth(selectionSet ast.SelectionSet) int {
	if selectionSet == nil {
		return 0
	}

	depth := 1

	for _, selection := range selectionSet {
		switch v := selection.(type) {
		case *ast.Field:
			selectionDepth := countDepth(v.SelectionSet) + 1
			if selectionDepth > depth {
				depth = selectionDepth
			}
		case *ast.FragmentSpread:
			selectionDepth := countDepth(v.Definition.SelectionSet)
			if selectionDepth > depth {
				depth = selectionDepth
			}
		}
	}
	return depth

}
