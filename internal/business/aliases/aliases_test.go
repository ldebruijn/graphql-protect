package aliases

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
	"github.com/vektah/gqlparser/v2/validator"
	"testing"
)

func Test_MaxAliasesRule(t *testing.T) {
	schema := `
type Query {
   getBook(title: String): Book
}

type Book {
	id: ID!
	title: String
	author: String
}`

	q := `query {
    firstBooks: getBook(title: "null") {
      author
      title
    }
    secondBooks: getBook(title: "null") {
      author
      title
    }
  }`

	type args struct {
		query  string
		schema string
		cfg    Config
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{
			name: "no aliases yields zero count",
			args: args{
				query:  q,
				schema: schema,
				cfg: Config{
					Max:     15,
					Enabled: true,
				},
			},
			want: nil,
		},
		{
			name: "does not produce error when counted aliases are more than configured maximum and reject on failure is false",
			args: args{
				cfg: Config{
					Enabled:         true,
					Max:             1,
					RejectOnFailure: false,
				},
				query:  q,
				schema: schema,
			},
			want: nil,
		},
		{
			name: "produces error when counted aliases are more than configured maximum and reject on failure is true",
			args: args{
				cfg: Config{
					Max:             1,
					Enabled:         true,
					RejectOnFailure: true,
				},
				query:  q,
				schema: schema,
			},
			want: fmt.Errorf("syntax error: Aliases limit of %d exceeded, found %d", 1, 2),
		},
		{
			name: "respects fragment aliases",
			args: args{
				query: `query A {
		 getBook(title: "null") {
		   firstTitle: title
		   ...BookFragment
		 }
		}
		fragment BookFragment on Book {
		 secondTitle: title
		}`,
				schema: schema,
				cfg: Config{
					Max:             1,
					Enabled:         true,
					RejectOnFailure: true,
				},
			},
			want: fmt.Errorf("syntax error: Aliases limit of %d exceeded, found %d", 1, 2),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			NewMaxAliasesRule(tt.args.cfg)

			query, _ := parser.ParseQuery(&ast.Source{Name: "ff", Input: tt.args.query})
			schema := gqlparser.MustLoadSchema(&ast.Source{
				Name:    "graph/schema.graphqls",
				Input:   tt.args.schema,
				BuiltIn: false,
			})

			errs := validator.Validate(schema, query)

			if tt.want == nil {
				assert.Empty(t, errs)
			} else {
				assert.Equal(t, tt.want.Error(), errs[0].Message)
			}
			errs = nil
		})
	}
}
