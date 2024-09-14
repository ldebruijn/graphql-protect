package max_depth // nolint:revive

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/validator"
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
		Enabled: false,
		Max:     15,
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

func NewMaxDepthRule(log *slog.Logger, cfg Config) { // nolint:funlen,cyclop // to be cleaned up after deprecated configuration fields are removed
	if cfg.Max != cfg.Field.Max {
		log.Warn("Using old `max_depth` configuration. Please update to new configuration options, see https://github.com/ldebruijn/graphql-protect/blob/main/docs/protections/max_depth.md")
	}
	if cfg.Enabled && cfg.Field.Enabled {
		// if both old and new config options are supplied, disable the old to prevent doing it twice
		cfg.Enabled = false
	}

	validator.AddRule("MaxDepth", func(observers *validator.Events, addError validator.AddErrFunc) {
		observers.OnOperation(func(_ *validator.Walker, operation *ast.OperationDefinition) {
			fieldDepth, listDepth := countDepth(operation.SelectionSet)

			if cfg.Field.Enabled {
				if fieldDepth > cfg.Field.Max {
					if cfg.Field.RejectOnFailure {
						addError(
							validator.Message("syntax error: Field depth limit of %d exceeded, found %d", cfg.Field.Max, fieldDepth),
							validator.At(operation.Position),
						)
						resultCounter.WithLabelValues("field", "rejected").Inc()
					} else {
						resultCounter.WithLabelValues("field", "failed").Inc()
					}
				} else {
					resultCounter.WithLabelValues("field", "allowed").Inc()
				}
			}

			if cfg.List.Enabled {
				if listDepth > cfg.List.Max {
					if cfg.List.RejectOnFailure {
						addError(
							validator.Message("syntax error: List depth limit of %d exceeded, found %d", cfg.List.Max, listDepth),
							validator.At(operation.Position),
						)
						resultCounter.WithLabelValues("list", "rejected").Inc()
					} else {
						resultCounter.WithLabelValues("list", "failed").Inc()
					}
				} else {
					resultCounter.WithLabelValues("list", "allowed").Inc()
				}
			}

			if cfg.Enabled {
				if fieldDepth > cfg.Max {
					if cfg.RejectOnFailure {
						addError(
							validator.Message("syntax error: Depth limit of %d exceeded, found %d", cfg.Max, fieldDepth),
							validator.At(operation.Position),
						)
						resultCounter.WithLabelValues("field", "rejected").Inc()
					} else {
						resultCounter.WithLabelValues("field", "failed").Inc()
					}
				} else {
					resultCounter.WithLabelValues("field", "allowed").Inc()
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
