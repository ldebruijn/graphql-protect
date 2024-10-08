package validation

import (
	"github.com/vektah/gqlparser/v2/gqlerror"
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
