package aliases

import (
	"fmt"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_MaxAliasesRule(t *testing.T) {
	query := `query {
    firstBooks: getBook(title: "null") {
      author
      title
    }
    secondBooks: getBook(title: "null") {
      author
      title
    }
  }`

	bookType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Book",
		Fields: graphql.Fields{
			"title":  &graphql.Field{Type: graphql.String},
			"author": &graphql.Field{Type: graphql.String},
		},
	})

	queryType := graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"books": &graphql.Field{
				Type: bookType,
			},
		},
	}

	type args struct {
		query  string
		schema graphql.ObjectConfig
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
				query:  query,
				schema: queryType,
				cfg: Config{
					Max: 15,
				},
			},
			want: nil,
		},
		{
			name: "produces error when counted aliases are more than configured maximum",
			args: args{
				cfg: Config{
					Max: 1,
				},
				query:  query,
				schema: queryType,
			},
			want: fmt.Errorf("syntax error: Aliases limit of %d exceeded, found %d", 1, 2),
		},
		{
			name: "respects fragment aliases",
			args: args{
				query: `
query A {
        getBook(title: "null") {
          firstTitle: title
          ...BookFragment
        }
      }
      fragment BookFragment on Book {
        secondTitle: title
      }
`,
				schema: queryType,
				cfg: Config{
					Max: 1,
				},
			},
			want: fmt.Errorf("syntax error: Aliases limit of %d exceeded, found %d", 1, 2),
		},
		{
			name: "does not crash on recursive fragments",
			args: args{
				query: `
query {
        ...A
      }

      fragment A on Query {
        ...B
      }

      fragment B on Query {
        ...A
      }
`,
				schema: func() graphql.ObjectConfig {
					aFragment := graphql.NewObject(graphql.ObjectConfig{
						Name: "A",
						//Fields: graphql.Fields{
						//	"B": &graphql.Field{Type: bFragment},
						//},
					})

					bFragment := graphql.NewObject(graphql.ObjectConfig{
						Name: "B",
						Fields: graphql.Fields{
							"A": &graphql.Field{Type: aFragment},
						},
					})

					aFragment.AddFieldConfig("B", &graphql.Field{Type: bFragment})

					queryType := graphql.ObjectConfig{
						Name: "Query",
						Fields: graphql.Fields{
							"A": &graphql.Field{
								Type: aFragment,
							},
						},
					}
					return queryType
				}(),
				cfg: Config{
					Max: 3,
				},
			},
			want: fmt.Errorf("Cannot spread fragment \"%s\" within itself via %s.", "A", "B"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			astDoc := parseQuery(t, tt.args.query)
			schema, _ := graphql.NewSchema(graphql.SchemaConfig{Query: graphql.NewObject(tt.args.schema)})

			ma := NewMaxAliasesRule(tt.args.cfg)

			vr := graphql.ValidateDocument(&schema, astDoc, []graphql.ValidationRuleFn{graphql.NoFragmentCyclesRule, ma.Validate})
			errs := vr.Errors
			if tt.want != nil {
				assert.Equal(t, tt.want.Error(), errs[0].Message)
			}
		})
	}
}

func parseQuery(t *testing.T, q string) *ast.Document {
	t.Helper()
	astDoc, err := parser.Parse(parser.ParseParams{Source: q})
	if err != nil {
		t.Fatalf("parse failed: %s", err)
	}
	return astDoc
}
