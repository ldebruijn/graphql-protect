package validation

import (
	"github.com/vektah/gqlparser/v2/gqlerror"
)

type ValidationError struct {
	Hash      string         `json:"-"`
	Operation string         `json:"-"`
	Err       gqlerror.Error `json:"-"`
}

func (v ValidationError) Error() string {
	if v.Hash != "" {
		return "Error valiating hash [" + v.Hash + "]: " + v.Err.Message
	}
	return v.Err.Message
}

func Wrap(err error) ValidationError {
	return ValidationError{
		Err: *gqlerror.Wrap(err),
	}
}
