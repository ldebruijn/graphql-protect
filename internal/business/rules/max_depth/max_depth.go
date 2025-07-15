package max_depth // nolint:revive

import (
	"fmt"
	"github.com/ldebruijn/graphql-protect/internal/business/validation"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/validator"
	"github.com/vektah/gqlparser/v2/validator/core"
	validatorrules "github.com/vektah/gqlparser/v2/validator/rules"
	"log/slog"
)

var resultCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "graphql_protect",
	Subsystem: "max_depth",
	Name:      "results",
	Help:      "The results of the max_depth rule",
},
	[]string{"type", "result"},
)

type Config struct {
	Enabled         bool    `yaml:"enabled"` // deprecated
	Max             int     `yaml:"max"`     // deprecated
	Field           MaxRule `yaml:"field"`
	List            MaxRule `yaml:"list"`
	RejectOnFailure bool    `yaml:"reject_on_failure"` // deprecated
}

type MaxRule struct {
	Enabled         bool `yaml:"enabled"`
	Max             int  `yaml:"max"`
	RejectOnFailure bool `yaml:"reject_on_failure"`
}

func DefaultConfig() Config {
	return Config{
		Field: MaxRule{
			Enabled:         true,
			Max:             15,
			RejectOnFailure: true,
		},
		List: MaxRule{
			Enabled:         true,
			Max:             2,
			RejectOnFailure: true,
		},
		RejectOnFailure: false,
	}
}

func init() {
	prometheus.MustRegister(resultCounter)
}

func NewMaxDepthRule(log *slog.Logger, cfg Config, rules *validatorrules.Rules) { // nolint:funlen,cyclop // to be cleaned up after deprecated configuration fields are removed
	rules.AddRule("MaxDepth", func(observers *validator.Events, addError core.AddErrFunc) {
		observers.OnOperation(func(_ *validator.Walker, operation *ast.OperationDefinition) {
			fieldDepth, listDepth := countDepth(operation.SelectionSet)

			if cfg.Field.Enabled {
				if fieldDepth > cfg.Field.Max {
					if cfg.Field.RejectOnFailure {
						addError(validation.RuleValidationResult{
							Rule:          "max-depth",
							OperationName: operation.Name,
							Result:        validation.REJECTED,
							Message:       fmt.Sprintf("field depth limit of %d exceeded, found %d", cfg.Field.Max, fieldDepth),
						}.Wrap())
						resultCounter.WithLabelValues("field", "rejected").Inc()
					} else {
						addError(validation.RuleValidationResult{
							Rule:          "max-depth",
							OperationName: operation.Name,
							Result:        validation.FAILED,
							Message:       fmt.Sprintf("field depth limit of %d exceeded, found %d", cfg.Field.Max, fieldDepth),
						}.Wrap())
						resultCounter.WithLabelValues("field", "failed").Inc()
					}
				} else {
					resultCounter.WithLabelValues("field", "allowed").Inc()
				}
			}

			if cfg.List.Enabled {
				if listDepth > cfg.List.Max {
					if cfg.List.RejectOnFailure {
						addError(validation.RuleValidationResult{
							Rule:          "max-depth",
							OperationName: operation.Name,
							Result:        "REJECTED",
							Message:       fmt.Sprintf("list depth limit of %d exceeded, found %d", cfg.List.Max, listDepth),
						}.Wrap())
						resultCounter.WithLabelValues("list", "rejected").Inc()
					} else {
						addError(validation.RuleValidationResult{
							Rule:          "max-depth",
							OperationName: operation.Name,
							Result:        validation.FAILED,
							Message:       fmt.Sprintf("list depth limit of %d exceeded, found %d", cfg.List.Max, listDepth),
						}.Wrap())
						resultCounter.WithLabelValues("list", "failed").Inc()
					}
				} else {
					resultCounter.WithLabelValues("list", "allowed").Inc()
				}
			}
		})
	})
}

func countDepth(selectionSet ast.SelectionSet) (int, int) { // nolint:cyclop // inherently cyclomatic
	if selectionSet == nil {
		return 0, 0
	}

	// start with 1 depth because root counts as the first depth
	fieldDepth := 1
	// start with 0 depth because we don't know yet if it is a list type
	listDepth := 0

	for _, selection := range selectionSet {
		switch v := selection.(type) {
		case *ast.Field:
			fieldSelectionDepth, listSelectionDepth := countDepth(v.SelectionSet)
			fieldSelectionDepth++ // increase because we're on a field

			if v.Definition != nil && isList(v.Definition.Type) {
				listSelectionDepth++ // increase because we're on a list
			}
			if listSelectionDepth > listDepth {
				listDepth = listSelectionDepth
			}
			if fieldSelectionDepth > fieldDepth {
				fieldDepth = fieldSelectionDepth
			}
		case *ast.FragmentSpread:
			fieldSelectionDepth, listSelectionDepth := countDepth(v.Definition.SelectionSet)
			if fieldSelectionDepth > fieldDepth {
				fieldDepth = fieldSelectionDepth
			}
			if listSelectionDepth > listDepth {
				listDepth = listSelectionDepth
			}
		}
	}
	return fieldDepth, listDepth

}

func isList(t *ast.Type) bool {
	if t == nil {
		return false
	}

	if t.NamedType != "" {
		return false
	}
	return true
}
