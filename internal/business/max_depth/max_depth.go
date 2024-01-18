package max_depth

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/validator"
)

var resultCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "go_graphql_armor",
	Subsystem: "max_depth",
	Name:      "results",
	Help:      "The results of the max_depth rule",
},
	[]string{"result"},
)

type Config struct {
	Enabled         bool `conf:"default:true" yaml:"enabled"`
	Max             int  `conf:"default:15" yaml:"max"`
	RejectOnFailure bool `conf:"default:true" yaml:"reject_on_failure"`
}

func init() {
	prometheus.MustRegister(resultCounter)
}

func NewMaxDepthRule(cfg Config) {
	if cfg.Enabled {
		validator.AddRule("MaxDepth", func(observers *validator.Events, addError validator.AddErrFunc) {
			observers.OnOperation(func(walker *validator.Walker, operation *ast.OperationDefinition) {
				var maxDepth = countDepth(operation.SelectionSet)

				if maxDepth > cfg.Max {
					if cfg.RejectOnFailure {
						err := fmt.Sprintf("syntax error: Depth limit of %d exceeded, found %d", cfg.Max, maxDepth)
						addError(
							validator.Message(err),
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
