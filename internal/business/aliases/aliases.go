package aliases

import (
	"fmt"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/kinds"
	"github.com/graphql-go/graphql/language/visitor"
	"sync"
)

type Config struct {
	max int `conf:"default:15" yaml:"max"`
}

var addRule sync.Once

type MaxAliasesRule struct {
	cfg Config
}

func NewMaxAliases() (*MaxAliasesRule, error) {

	addRule.Do(func() {
		r := MaxAliasesRule{}
		graphql.SpecifiedRules = append(graphql.SpecifiedRules, r.validate)
	})
	return nil, nil
}

func (a *MaxAliasesRule) validate(context *graphql.ValidationContext) *graphql.ValidationRuleInstance {
	instance := a.newMaxRuleInstance(a.cfg, context)
	return &graphql.ValidationRuleInstance{VisitorOpts: instance.visitorOptions()}
}

type maxAliasesRuleInstance struct {
	visitedFragments  map[string]int
	cfg               Config
	validationContext *graphql.ValidationContext
}

func (a *MaxAliasesRule) newMaxRuleInstance(cfg Config, validationContext *graphql.ValidationContext) *maxAliasesRuleInstance {
	return &maxAliasesRuleInstance{
		visitedFragments:  map[string]int{},
		cfg:               cfg,
		validationContext: validationContext,
	}
}

func (a *maxAliasesRuleInstance) visitorOptions() *visitor.VisitorOptions {
	return &visitor.VisitorOptions{
		KindFuncMap: map[string]visitor.NamedVisitFuncs{
			kinds.OperationDefinition: {
				Enter: a.onOperationDefinitionEnter,
				//Leave: a.onOperationDefinitionLeave,
			},
		},
	}
}

func (i *maxAliasesRuleInstance) onOperationDefinitionEnter(p visitor.VisitFuncParams) (string, interface{}) {
	od, ok := p.Node.(*ast.OperationDefinition)
	if !ok {
		return visitor.ActionSkip, nil
	}

	aliases := i.countAliases(p.Node)

	if aliases > i.cfg.max {
		err := fmt.Sprintf("syntax Error: Aliases limit of %d exceeded, found %d", i.cfg.max, aliases)

		i.validationContext.ReportError(gqlerrors.NewError(err, []ast.Node{od}, "", nil, []int{}, nil))
	}
	return visitor.ActionNoChange, nil
}

func (i *maxAliasesRuleInstance) countAliases(node interface{}) int {
	aliases := 0

	switch node.(type) {
	case ast.Field:
		if node.(ast.Field).Alias != nil {
			aliases++
		}
	case ast.SelectionSet:
		for _, child := range node.(ast.SelectionSet).Selections {
			aliases += i.countAliases(child)
		}
	case ast.FragmentSpread:
		value := node.(ast.FragmentSpread).Name.Value
		if val, ok := i.visitedFragments[value]; ok {
			return val
		} else {
			i.visitedFragments[value] = -1
		}

		if fragment := i.validationContext.Fragment(value); fragment != nil {
			additionalAliases := i.countAliases(fragment)
			if i.visitedFragments[value] == -1 {
				i.visitedFragments[value] = additionalAliases
			}
			aliases += additionalAliases
		}
	}

	return aliases
}
