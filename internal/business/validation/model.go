package validation

import (
	"github.com/vektah/gqlparser/v2/gqlerror"
)

type Error struct {
	Hash      string         `json:"-"`
	Operation string         `json:"-"`
	Err       gqlerror.Error `json:"-"`
}

func (v Error) Error() string {
	if v.Hash != "" {
		return "Error valiating hash [" + v.Hash + "]: " + v.Err.Message
	}
	return v.Err.Message
}

func Wrap(err error) Error {
	return Error{
		Err: *gqlerror.Wrap(err),
	}
}
