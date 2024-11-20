package trusteddocuments

import (
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
		want string
		err  error
	}{
		{
			name: "extracts operation from query",
			args: args{
				operation: "query ProductQuery{ product(id: 1) { id title as } }",
			},
			want: "ProductQuery",
		},
		{
			name: "extracts operation from mutation",
			args: args{
				operation: "mutation ProductQuery{ product(id: 1) { id title as } }",
			},
			want: "ProductQuery",
		},
		{
			name: "no operation name when not present",
			args: args{
				operation: "mutation { product(id: 1) { id title as } }",
			},
			want: "",
		},
		{
			name: "no operation name when no space between type and bracket",
			args: args{
				operation: "mutation{ product(id: 1) { id title as } }",
			},
			want: "",
		},
		{
			name: "excludes operation arguments",
			args: args{
				operation: "query Foobar($some: Int, $value: String){ product(id: 1) { id title as } }",
			},
			want: "Foobar",
		},
		{
			name: "no weird stuff when getting a completely malformed string",
			args: args{
				operation: "",
			},
			want: "",
		},
		{
			name: "handles white space around operation name",
			args: args{
				operation: "query Foobar ($some: Int, $value: String){ product(id: 1) { id title as } }",
			},
			want: "Foobar",
		},
		{
			name: "error is thrown on non parseable queries",
			args: args{
				operation: "invalidQueryString",
			},
			want: "",
		},
		{
			name: "Can deal with fragments inside a query",
			args: args{
				operation: "fragment BaseItem on MenuItem { action { menuItemActionType url } id imageUrl level measurement { clickedMenuItem } title } query MenuItems($country: String!, $id: String, $language: String!, $levels: String) { menuItems(country: $country, language: $language, id: $id, levels: $levels) { children { children { ...BaseItem } ...BaseItem } imageHeaderUrl level ...BaseItem } }",
			},
			want: "MenuItems",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			operation := extractOperationNameFromOperation(tt.args.operation)
			assert.Equalf(t, tt.want, operation, "newPersistedOperation(%v)", tt.args.operation)
		})
	}
}
