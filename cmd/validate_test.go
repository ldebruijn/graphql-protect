package main

import (
	"bytes"
	"errors"
	"github.com/ldebruijn/graphql-protect/internal/business/validation"
	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"testing"
)

func Test_formatErrors(t *testing.T) {
	type args struct {
		errs []validation.Error
		w    *bytes.Buffer
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "no errors no content",
			args: args{
				errs: make([]validation.Error, 0),
				w:    &bytes.Buffer{},
			},
			want: `+-------+------+------+-------+
|     # | HASH | RULE | ERROR |
+-------+------+------+-------+
+-------+------+------+-------+
| TOTAL |    0 |      |       |
+-------+------+------+-------+
`,
		},
		{
			name: "error is present in table",
			args: args{
				errs: []validation.Error{
					{
						Hash:      "i am a hash",
						Operation: "",
						Err: gqlerror.Error{
							Err:        errors.New("ohoh"),
							Message:    "something went wrong",
							Path:       nil,
							Locations:  nil,
							Extensions: nil,
							Rule:       "foobar",
						},
					},
				},
				w: &bytes.Buffer{},
			},
			want: `+-------+-------------+--------+----------------------+
|     # | HASH        | RULE   | ERROR                |
+-------+-------------+--------+----------------------+
|     0 | i am a hash | foobar | something went wrong |
+-------+-------------+--------+----------------------+
| TOTAL | 1           |        |                      |
+-------+-------------+--------+----------------------+
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatErrors(tt.args.w, tt.args.errs)
			assert.Equalf(t, tt.want, tt.args.w.String(), "formatErrors(%v, %v)", tt.args.w, tt.args.errs)
		})
	}
}
