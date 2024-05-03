package max_depth // nolint:revive

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
	"github.com/vektah/gqlparser/v2/validator"
	"testing"
)

func Test_MaxDepthRule(t *testing.T) {
	schema := `
type Query {
   getBook(title: String): Book
}

type Book {
	id: ID!
	title: String
	author: Author!
	price: Price!
}
type Author {
	id: ID!
	name: String
}
type Price {
	price: Int!
	id: ID!
}
`
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
			name: "no query yields zero count",
			args: args{
				query:  "",
				schema: schema,
				cfg: Config{
					Max:     15,
					Enabled: true,
				},
			},
			want: nil,
		},
		{
			name: "Calculate the depth properly with fragments",
			args: args{
				cfg: Config{
					Max:             3,
					Enabled:         true,
					RejectOnFailure: true,
				},
				query: `
				query A {
					getBook(title: "null") {
						id
				   		...BookFragment
				 	}
				}
				fragment BookFragment on Book {
				 	author {
						name
					}
				}`,
				schema: schema,
			},
			want: nil,
		},
		{
			name: "Calculate depth properly",
			args: args{
				cfg: Config{
					Enabled:         true,
					Max:             2,
					RejectOnFailure: true,
				},
				query: `
					query {
						getBook(title: "null") {
						  title
						  price {
							price
							id
						  }
						}
					}`,
				schema: schema,
			},
			want: fmt.Errorf("syntax error: Depth limit of %d exceeded, found %d", 2, 3),
		},
		{
			name: "Works correctly with fragments",
			args: args{
				cfg: Config{
					Max:             2,
					Enabled:         true,
					RejectOnFailure: true,
				},
				query: `
				query A {
					getBook(title: "null") {
						id
				   		...BookFragment
				 	}
				}
				fragment BookFragment on Book {
				 	author {
						name
					}
				}`,
				schema: schema,
			},
			want: fmt.Errorf("syntax error: Depth limit of %d exceeded, found %d", 2, 3),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			NewMaxDepthRule(tt.args.cfg)

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
		})
	}
}
