package tokens

import (
	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/v2/ast"
	"testing"
)

func TestMaxTokens(t *testing.T) {
	type args struct {
		cfg           Config
		operation     string
		operationName string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "rule disabled does nothing",
			args: args{
				cfg: Config{
					Enabled:         false,
					Max:             0,
					RejectOnFailure: true,
				},
				operation: "query { foo }",
			},
			wantErr: false,
		},
		{
			name: "yields error when tokens exceed max",
			args: args{
				cfg: Config{
					Enabled:         true,
					Max:             1,
					RejectOnFailure: true,
				},
				operation: "query { foo }",
			},
			wantErr: true,
		},
		{
			name: "yields no error when tokens less than max",
			args: args{
				cfg: Config{
					Enabled:         true,
					Max:             1000000,
					RejectOnFailure: true,
				},
				operation: "query { foo }",
			},
			wantErr: false,
		},
		{
			name: "yields no error when tokens exceed max but failure on rejections is false",
			args: args{
				cfg: Config{
					Enabled:         true,
					Max:             1,
					RejectOnFailure: false,
				},
				operation: "query { foo }",
			},
			wantErr: false,
		},
		{
			name: "override is respected",
			args: args{
				cfg: Config{
					Enabled:         true,
					Max:             1,
					RejectOnFailure: true,
					Overrides: map[string]int{
						"MyOperation": 100,
					},
				},
				operation:     "query MyOperation { foo }",
				operationName: "MyOperation",
			},
			wantErr: false,
		},
		{
			name: "override is respected and fails if limit is exceeded",
			args: args{
				cfg: Config{
					Enabled:         true,
					Max:             100,
					RejectOnFailure: true,
					Overrides: map[string]int{
						"MyOperation": 1,
					},
				},
				operation:     "query MyOperation { foo }",
				operationName: "MyOperation",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := MaxTokens(tt.args.cfg)

			source := &ast.Source{
				Input: tt.args.operation,
			}

			err := rule.Validate(source, tt.args.operationName)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
