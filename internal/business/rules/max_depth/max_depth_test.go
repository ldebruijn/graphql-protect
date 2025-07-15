package max_depth // nolint:revive

import (
	"fmt"
	"github.com/ldebruijn/graphql-protect/internal/business/validation"
	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/vektah/gqlparser/v2/parser"
	"github.com/vektah/gqlparser/v2/validator"
	validatorrules "github.com/vektah/gqlparser/v2/validator/rules"
	"testing"
)

func Test_MaxListDepthRule(t *testing.T) {
	schema := `
type Query {
   me: User
}

type User {
	id: ID!
	name: String
	friends: [User!]!
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
					List: MaxRule{
						Max:     15,
						Enabled: true,
					},
				},
			},
			want: nil,
		},
		{
			name: "Calculate the depth properly with fragments",
			args: args{
				cfg: Config{
					List: MaxRule{
						Max:             3,
						Enabled:         true,
						RejectOnFailure: true,
					},
				},
				query: `
				query A {
					me {
						...UserFragment
				 	}
				}
				fragment UserFragment on User {
				 	id
					name
				}`,
				schema: schema,
			},
			want: nil,
		},
		{
			name: "Calculate list depth properly",
			args: args{
				cfg: Config{
					Field: MaxRule{
						Enabled: false,
					},
					List: MaxRule{
						Enabled:         true,
						Max:             2,
						RejectOnFailure: true,
					},
				},
				query: `
				query A {
					me {
						id
						name
						friends {
							name
							friends {
								name
								friends {
									name
									friends {
										name
									}
								}
							}
						}
				 	}
				}`,
				schema: schema,
			},
			want: fmt.Errorf("list depth limit of %d exceeded, found %d", 2, 4),
		},
		{
			name: "Calculates list depth per nested list. Does not sum counts of each list",
			args: args{
				cfg: Config{
					Field: MaxRule{
						Enabled: false,
					},
					List: MaxRule{
						Enabled:         true,
						Max:             2,
						RejectOnFailure: true,
					},
				},
				query: `
				query A {
					a1: me {
						id
						name
						friends {
							name
							friends {
								name
							}
						}
				 	}
					a2: me {
						id
						name
						friends {
							name
							friends {
								name
							}
						}
				 	}
				}`,
				schema: schema,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := validatorrules.NewDefaultRules()

			NewMaxDepthRule(tt.args.cfg, rules)

			query, _ := parser.ParseQuery(&ast.Source{Name: "ff", Input: tt.args.query})
			schema := gqlparser.MustLoadSchema(&ast.Source{
				Name:    "graph/schema.graphqls",
				Input:   tt.args.schema,
				BuiltIn: false,
			})

			errs := validator.ValidateWithRules(schema, query, rules)

			if tt.want == nil {
				assert.Empty(t, errs)
			} else {
				assert.Equal(t, tt.want.Error(), errs[0].Message)
			}

			validator.RemoveRule("MaxDepth")
		})
	}
}

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
		want *gqlerror.Error
	}{
		{
			name: "no query yields zero count",
			args: args{
				query:  "",
				schema: schema,
				cfg: Config{
					Field: MaxRule{
						Max:     15,
						Enabled: true,
					},
				},
			},
			want: nil,
		},
		{
			name: "Calculate the depth properly with fragments",
			args: args{
				cfg: Config{
					Field: MaxRule{
						Max:             3,
						Enabled:         true,
						RejectOnFailure: true,
					},
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
					Field: MaxRule{
						Enabled:         true,
						Max:             2,
						RejectOnFailure: true,
					},
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
			want: validation.RuleValidationResult{
				Rule:          "max-aliases",
				OperationName: "",
				Result:        validation.REJECTED,
				Message:       fmt.Sprintf("field depth limit of %d exceeded, found %d", 2, 3),
			}.AsGqlError(),
		},
		{
			name: "Works correctly with fragments",
			args: args{
				cfg: Config{
					Field: MaxRule{
						Max:             2,
						Enabled:         true,
						RejectOnFailure: true,
					},
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
			want: validation.RuleValidationResult{
				Rule:          "max-aliases",
				OperationName: "",
				Result:        validation.REJECTED,
				Message:       fmt.Sprintf("field depth limit of %d exceeded, found %d", 2, 3),
			}.AsGqlError(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := validatorrules.NewDefaultRules()

			NewMaxDepthRule(tt.args.cfg, rules)

			query, _ := parser.ParseQuery(&ast.Source{Name: "ff", Input: tt.args.query})
			schema := gqlparser.MustLoadSchema(&ast.Source{
				Name:    "graph/schema.graphqls",
				Input:   tt.args.schema,
				BuiltIn: false,
			})

			errs := validator.ValidateWithRules(schema, query, rules)

			if tt.want == nil {
				assert.Empty(t, errs)
			} else {
				assert.Equal(t, tt.want.Message, errs[0].Message)
			}

			validator.RemoveRule("MaxDepth")
		})
	}
}
