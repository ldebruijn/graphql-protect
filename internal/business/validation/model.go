package validation

import (
	"fmt"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/vektah/gqlparser/v2/validator/core"
)

type Error struct {
	Hash string         `json:"-"`
	Err  gqlerror.Error `json:"-"`
}

func (v Error) Error() string {
	if v.Hash != "" {
		return "Error validating hash [" + v.Hash + "]: " + v.Err.Error()
	}
	return v.Err.Message
}

func Wrap(err error) Error {
	return Error{
		Err: *gqlerror.Wrap(err),
	}
}

type Result string

const (
	FAILED   Result = "FAILED"
	REJECTED Result = "REJECTED"
	ALLOWED  Result = "ALLOWED"
)

type RuleValidationResult struct {
	Rule          string
	OperationName string
	Result        Result
	Message       string
}

func (r RuleValidationResult) Error() string {
	return fmt.Sprintf("%s: operation %s validation %s: %s", r.Rule, r.OperationName, r.Result, r.Message)
}

func (r RuleValidationResult) AsGqlError() *gqlerror.Error {
	return &gqlerror.Error{
		Err:        r,
		Message:    r.Message,
		Path:       nil,
		Locations:  nil,
		Extensions: nil,
		Rule:       r.Rule,
	}
}

func (r RuleValidationResult) Wrap() core.ErrorOption {
	return func(e *gqlerror.Error) {
		g := r.AsGqlError()
		e.Message = g.Message
		e.Err = g.Err
		e.Rule = g.Rule
	}
}
