package persistedoperations

import (
	"errors"
	"github.com/stretchr/testify/assert"
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
		err  error
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
		{
			name: "excludes operation arguments",
			args: args{
				operation: "query Foobar($some: Int, $value: String){ product(id: 1) { id title as } }",
			},
			want: PersistedOperation{
				Operation: "query Foobar($some: Int, $value: String){ product(id: 1) { id title as } }",
				Name:      "Foobar",
			},
		},
		{
			name: "no weird stuff when getting a completely malformed string",
			args: args{
				operation: "",
			},
			want: PersistedOperation{
				Operation: "",
				Name:      "",
			},
		},
		{
			name: "handles white space around operation name",
			args: args{
				operation: "query Foobar ($some: Int, $value: String){ product(id: 1) { id title as } }",
			},
			want: PersistedOperation{
				Operation: "query Foobar ($some: Int, $value: String){ product(id: 1) { id title as } }",
				Name:      "Foobar",
			},
		},
		{
			name: "error is thrown on non parseable queries",
			args: args{
				operation: "invalidQueryString",
			},
			want: PersistedOperation{
				Operation: "invalidQueryString",
				Name:      "",
			},
			err: errors.New("no operation name match found for query/mutation invalidQueryString"),
		},
		{
			name: "Can deal with fragments inside a query",
			args: args{
				operation: "fragment BaseItem on MenuItem { action { menuItemActionType url } id imageUrl level measurement { clickedMenuItem } title } query MenuItems($country: String!, $id: String, $language: String!, $levels: String) { menuItems(country: $country, language: $language, id: $id, levels: $levels) { children { children { ...BaseItem } ...BaseItem } imageHeaderUrl level ...BaseItem } }",
			},
			want: PersistedOperation{
				Operation: "fragment BaseItem on MenuItem { action { menuItemActionType url } id imageUrl level measurement { clickedMenuItem } title } query MenuItems($country: String!, $id: String, $language: String!, $levels: String) { menuItems(country: $country, language: $language, id: $id, levels: $levels) { children { children { ...BaseItem } ...BaseItem } imageHeaderUrl level ...BaseItem } }",
				Name:      "MenuItems",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			operation := extractOperationNameFromOperation(tt.args.operation)
			assert.Equalf(t, tt.want, operation, "newPersistedOperation(%v)", tt.args.operation)
		})
	}
}
