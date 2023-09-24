package block_field_suggestions

import (
	"reflect"
	"testing"
)

func TestProcessBody(t *testing.T) {
	type args struct {
		payload map[string]interface{}
	}
	tests := []struct {
		name string
		args args
		want map[string]interface{}
	}{
		{
			name: "nothing happens when not expected format",
			args: args{
				payload: map[string]interface{}{
					"hi": "bye",
				},
			},
			want: map[string]interface{}{
				"hi": "bye",
			},
		},
		{
			name: "processes errors payload",
			args: args{
				payload: map[string]interface{}{
					"errors": []map[string]interface{}{
						{
							"message": "hello",
						},
					},
				},
			},
			want: map[string]interface{}{
				"errors": []map[string]interface{}{
					{
						"message": "hello",
					},
				},
			},
		},
		{
			name: "can handle unexpected types",
			args: args{
				payload: map[string]interface{}{
					"errors": []map[string]interface{}{
						{
							"message": 1,
						},
					},
				},
			},
			want: map[string]interface{}{
				"errors": []map[string]interface{}{
					{
						"message": 1,
					},
				},
			},
		},
		{
			name: "Replaces suggestions when found",
			args: args{
				payload: map[string]interface{}{
					"errors": []map[string]interface{}{
						{
							"message": "Did you mean 'foobar'?",
						},
					},
				},
			},
			want: map[string]interface{}{
				"errors": []map[string]interface{}{
					{
						"message": "[redacted]",
					},
				},
			},
		},
		{
			name: "Doesn't affect any other fields",
			args: args{
				payload: map[string]interface{}{
					"data": map[string]interface{}{
						"foo":     "bar",
						"boolean": 1,
					},
					"errors": []map[string]interface{}{
						{
							"message":   "Did you mean 'foobar'?",
							"something": "else",
						},
						{
							"without": "message",
						},
					},
				},
			},
			want: map[string]interface{}{
				"data": map[string]interface{}{
					"foo":     "bar",
					"boolean": 1,
				},
				"errors": []map[string]interface{}{
					{
						"message":   "[redacted]",
						"something": "else",
					},
					{
						"without": "message",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBlockFieldSuggestionsHandler(Config{
				Enabled: true,
				Mask:    "[redacted]",
			})

			if got := b.ProcessBody(tt.args.payload); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ProcessBody() = %v, want %v", got, tt.want)
			}
		})
	}
}
