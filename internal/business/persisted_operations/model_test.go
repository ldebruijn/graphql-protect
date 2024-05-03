package persisted_operations

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestNewPersistedOperation(t *testing.T) {
	type args struct {
		operation string
	}
	tests := []struct {
		name string
		args args
		want PersistedOperation
	}{
		{
			name: "extracts operation from query",
			args: args{
				operation: "query ProductQuery{ product(id: 1) { id title as } }",
			},
			want: PersistedOperation{
				Operation: "query ProductQuery{ product(id: 1) { id title as } }",
				Name:      "ProductQuery",
			},
		},
		{
			name: "extracts operation from mutation",
			args: args{
				operation: "mutation ProductQuery{ product(id: 1) { id title as } }",
			},
			want: PersistedOperation{
				Operation: "mutation ProductQuery{ product(id: 1) { id title as } }",
				Name:      "ProductQuery",
			},
		},
		{
			name: "no operation name when not present",
			args: args{
				operation: "mutation { product(id: 1) { id title as } }",
			},
			want: PersistedOperation{
				Operation: "mutation { product(id: 1) { id title as } }",
				Name:      "",
			},
		},
		{
			name: "no operation name when no space between type and bracket",
			args: args{
				operation: "mutation{ product(id: 1) { id title as } }",
			},
			want: PersistedOperation{
				Operation: "mutation{ product(id: 1) { id title as } }",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, NewPersistedOperation(tt.args.operation), "NewPersistedOperation(%v)", tt.args.operation)
		})
	}
}

func TestFoo(t *testing.T) {
	type args struct {
		operation string
	}
	tests := []struct {
		name string
		args args
		want string
	}{}
	for _, tt := range tests {
		result := func(query string) string {
			firstSpace := strings.Index(query, " ")
			firstBracket := strings.Index(query, "{")

			if firstSpace > firstBracket {
				return ""
			}

			return query[firstSpace+1 : firstBracket]
		}(tt.args.operation)

		assert.Equal(t, tt.want, result, tt.name)
	}
}
