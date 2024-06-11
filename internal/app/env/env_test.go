package env

import (
	"reflect"
	"testing"
)

func Test_detect(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want Environment
	}{
		{
			name: "none specified",
			args: args{
				value: "pro",
			},
			want: Pro,
		},
		{
			name: "explicit dev",
			args: args{
				value: "dev",
			},
			want: Dev,
		},
		{
			name: "explicit pro",
			args: args{
				value: "pro",
			},
			want: Pro,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detect(tt.args.value); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("detect() = %v, want %v", got, tt.want)
			}
		})
	}
}
