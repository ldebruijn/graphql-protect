package aliases

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/validator"
)

var (
	resultCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "graphql_protect",
		Subsystem: "max_aliases",
		Name:      "results",
		Help:      "The results of the max aliases rule",
	},
		[]string{"result"},
	)
)

type Config struct {
	Enabled         bool `conf:"default:true" yaml:"enabled"`
	Max             int  `conf:"default:15" yaml:"max"`
	RejectOnFailure bool `conf:"default:true" yaml:"reject_on_failure"`
}

func init() {
	prometheus.MustRegister(resultCounter)
}

func NewMaxAliasesRule(cfg Config) {
	if cfg.Enabled {
		validator.AddRule("MaxAliases", func(observers *validator.Events, addError validator.AddErrFunc) {
			aliases := 0
			// keep track of # of aliases per fragment definition
			visitedFragments := make(map[string]int)

			observers.OnFragmentSpread(func(walker *validator.Walker, fragmentSpread *ast.FragmentSpread) {
				definition := fragmentSpread.Definition
				if _, ok := visitedFragments[definition.Name]; !ok {
					count := countSelectionSet(definition.SelectionSet)
					visitedFragments[definition.Name] = count
				}

				aliases += visitedFragments[definition.Name]
			})

			observers.OnOperation(func(walker *validator.Walker, operation *ast.OperationDefinition) {
				aliases += countAliases(operation)

				if aliases > cfg.Max {
					if cfg.RejectOnFailure {
						err := fmt.Sprintf("syntax error: Aliases limit of %d exceeded, found %d", cfg.Max, aliases)
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

func countAliases(operation *ast.OperationDefinition) int {
	return countSelectionSet(operation.SelectionSet)
}

func countSelectionSet(set ast.SelectionSet) int {
	count := 0
	if len(set) == 0 {
		return count
	}

	for _, selection := range set {
		if v, ok := selection.(*ast.Field); ok {
			// When a query has no alias defined it defaults to the name of the query
			if v.Alias != "" && v.Alias != v.Name {
				count++
			}

			count += countSelectionSet(v.SelectionSet)
		}
	}

	return count
}
