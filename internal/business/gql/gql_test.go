package gql

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestParseRequestPayload(t *testing.T) {
	type args struct {
		r *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    []RequestData
		wantErr bool
	}{
		{
			name: "parses regular operation correctly",
			args: args{
				r: func() *http.Request {
					payload := `
{
	"query": "something",
	"variables": {
		"baz": "foobar"	
	},
	"extensions": {
		"foo": "bar"	
	}
}
`
					body := bytes.NewBuffer([]byte(payload))
					return httptest.NewRequest("POST", "/graphql", body)
				}(),
			},
			want: []RequestData{
				{
					Variables: map[string]interface{}{
						"baz": "foobar",
					},
					Query:      "something",
					Extensions: Extensions{},
				},
			},
			wantErr: false,
		},
		{
			name: "parses batched operation correctly",
			args: args{
				r: func() *http.Request {
					payload := `
[
{
	"query": "query batched 1",
	"variables": {
		"baz": "variables batched 1"	
	},
	"extensions": {
		"foo": "extension batched 1"	
	}
},
{
	"query": "query batched 2",
	"variables": {
		"baz": "variables batched 2"	
	},
	"extensions": {
		"foo": "extensions batched 2"	
	}
}
]
`
					body := bytes.NewBuffer([]byte(payload))
					return httptest.NewRequest("POST", "/graphql", body)
				}(),
			},
			want: []RequestData{
				{
					Variables: map[string]interface{}{
						"baz": "variables batched 1",
					},
					Query:      "query batched 1",
					Extensions: Extensions{},
				},
				{
					Variables: map[string]interface{}{
						"baz": "variables batched 2",
					},
					Query:      "query batched 2",
					Extensions: Extensions{},
				},
			},
			wantErr: false,
		},
		{
			name: "Handles request without body gracefully",
			args: args{
				r: func() *http.Request {
					return httptest.NewRequest("POST", "/graphql", nil)
				}(),
			},
			want:    []RequestData{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRequestPayload(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRequestPayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseRequestPayload() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkCheckJSONType(b *testing.B) {
	// Create a sample JSON object
	jsonObject := []byte(`{
	"query": "something",
	"variables": {
		"baz": "foobar"	
	},
	"extensions": {
		"foo": "bar"	
	}
},`)

	// Create a sample JSON array
	jsonArray := []byte(fmt.Sprintf("[%[1]s, %[1]s, %[1]s, %[1]s, %[1]s]", jsonObject))

	for i := 0; i < b.N; i++ {
		// Benchmark decoding a JSON object
		b.Run("JSON Object", func(b *testing.B) {
			for j := 0; j < b.N; j++ {
				r := httptest.NewRequest("POST", "/graphql", bytes.NewBuffer(jsonObject))
				_, _ = ParseRequestPayload(r)
			}
		})

		// Benchmark decoding a JSON array
		b.Run("JSON Array", func(b *testing.B) {
			for j := 0; j < b.N; j++ {
				r := httptest.NewRequest("POST", "/graphql", bytes.NewBuffer(jsonArray))
				_, _ = ParseRequestPayload(r)
			}
		})
	}
}
