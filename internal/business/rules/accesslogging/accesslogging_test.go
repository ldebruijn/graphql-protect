package accesslogging

import (
	"context"
	"github.com/ldebruijn/graphql-protect/internal/business/gql"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"net/http"
	"testing"
)

type testLogHandler struct {
	assert func(ctx context.Context, record slog.Record) error
	count  int
	t      *testing.T
}

func (t *testLogHandler) setT(testing *testing.T) {
	t.t = testing
}

func (t *testLogHandler) Enabled(context.Context, slog.Level) bool {
	return true
}
func (t *testLogHandler) Handle(ctx context.Context, record slog.Record) error {
	t.count++
	return t.assert(ctx, record)
}
func (t *testLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return t
}
func (t *testLogHandler) WithGroup(name string) slog.Handler {
	return t
}

func TestAccessLogging_Log(t *testing.T) {
	type args struct {
		cfg      Config
		payloads []gql.RequestData
		headers  http.Header
		count    int
	}
	tests := []struct {
		name string
		args args
		want func(ctx context.Context, record slog.Record) error
	}{
		{
			name: "logs expected fields when enabled",
			args: args{
				cfg: Config{
					Enable:               true,
					IncludedHeaders:      []string{"Authorization"},
					IncludeOperationName: true,
					IncludeVariables:     true,
					IncludePayload:       true,
				},
				payloads: []gql.RequestData{
					{
						OperationName: "Foobar",
						Variables: map[string]interface{}{
							"foo": "bar",
						},
						Query: "query Foo { id name }",
					},
				},
				headers: map[string][]string{
					"Authorization": {"bearer hello"},
					"Content-Type":  {"application/json"},
				},
				count: 1,
			},
			want: func(ctx context.Context, record slog.Record) error {
				assert.Equal(t, 1, record.NumAttrs())
				record.Attrs(func(a slog.Attr) bool {
					assert.Equal(t, "payload", a.Key)
					val := a.Value.Any().(map[string]interface{})
					assert.Equal(t, "Foobar", val["operationName"])
					assert.Equal(t, "query Foo { id name }", val["payload"])
					assert.Equal(t, map[string]interface{}{
						"foo": "bar",
					}, val["variables"])
					assert.Equal(t, map[string]interface{}{
						"Authorization": []string{"bearer hello"},
					}, val["headers"])

					return true
				})

				return nil
			},
		},
		{
			name: "logs nothing when disabled",
			args: args{
				cfg: Config{
					Enable:               false,
					IncludedHeaders:      []string{"Authorization"},
					IncludeOperationName: true,
					IncludeVariables:     true,
					IncludePayload:       true,
				},
				payloads: []gql.RequestData{
					{
						OperationName: "Foobar",
						Variables: map[string]interface{}{
							"foo": "bar",
						},
						Query: "query Foo { id name }",
					},
				},
				headers: map[string][]string{
					"Authorization": {"bearer hello"},
					"Content-Type":  {"application/json"},
				},
				count: 0,
			},
			want: func(ctx context.Context, record slog.Record) error {
				assert.Fail(t, "should never reach here")
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &testLogHandler{assert: tt.want}
			log := slog.New(handler)

			a := &AccessLogging{
				log: log,
				cfg: tt.args.cfg,
			}
			a.Log(tt.args.payloads, tt.args.headers)

			assert.Equal(t, tt.args.count, a.log.Handler().(*testLogHandler).count)
		})
	}
}
