package batch

import (
	"github.com/ldebruijn/go-graphql-armor/internal/business/gql"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMaxBatchRule_Validate(t1 *testing.T) {
	type fields struct {
		cfg Config
	}
	type args struct {
		payload []gql.RequestData
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		initErr bool
	}{
		{
			name: "disabled has no effect",
			fields: fields{
				cfg: Config{
					Enabled:         false,
					Max:             1,
					RejectOnFailure: true,
				},
			},
			args: args{
				payload: []gql.RequestData{
					{
						Query:      "",
						Extensions: gql.Extensions{},
					},
					{
						Query:      "",
						Extensions: gql.Extensions{},
					},
					{
						Query:      "",
						Extensions: gql.Extensions{},
					},
				},
			},
			wantErr: false,
			initErr: false,
		},
		{
			name: "less than limit passes",
			fields: fields{
				cfg: Config{
					Enabled:         true,
					Max:             3,
					RejectOnFailure: true,
				},
			},
			args: args{
				payload: []gql.RequestData{
					{
						Query:      "",
						Extensions: gql.Extensions{},
					},
				},
			},
			wantErr: false,
			initErr: false,
		},
		{
			name: "more than limit throws",
			fields: fields{
				cfg: Config{
					Enabled:         true,
					Max:             1,
					RejectOnFailure: true,
				},
			},
			args: args{
				payload: []gql.RequestData{
					{
						Query:      "",
						Extensions: gql.Extensions{},
					},
					{
						Query:      "",
						Extensions: gql.Extensions{},
					},
					{
						Query:      "",
						Extensions: gql.Extensions{},
					},
				},
			},
			wantErr: true,
			initErr: false,
		},
		{
			name: "invalid config auto disables",
			fields: fields{
				cfg: Config{
					Enabled:         true,
					Max:             0,
					RejectOnFailure: true,
				},
			},
			args: args{
				payload: []gql.RequestData{
					{
						Query:      "",
						Extensions: gql.Extensions{},
					},
					{
						Query:      "",
						Extensions: gql.Extensions{},
					},
					{
						Query:      "",
						Extensions: gql.Extensions{},
					},
				},
			},
			wantErr: false,
			initErr: true,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t, err := NewMaxBatch(tt.fields.cfg)
			if tt.initErr {
				assert.Error(t1, err)
			}

			if err := t.Validate(tt.args.payload); (err != nil) != tt.wantErr {
				t1.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
