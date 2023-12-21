package aliases

import (
	"fmt"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/validator"
)

//var resultCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
//	Namespace: "go_graphql_armor",
//	Subsystem: "max_aliases",
//	Name:      "results",
//	Help:      "The results of the max aliases rule",
//},
//	[]string{"result"},
//)

type MaxAliasesRule2 struct {
	cfg Config
}

//func init() {
//	prometheus.MustRegister(resultCounter)
//}

func NewMaxAliasesRule2(cfg Config) {
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
		switch v := selection.(type) {
		case *ast.Field:
			if v.Alias != "" && v.Alias != v.Name {
				count++
			}

			count += countSelectionSet(v.SelectionSet)
		}
	}

	return count
}
