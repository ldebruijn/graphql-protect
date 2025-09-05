package accesslogging

import (
	"context"
	"log/slog"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ldebruijn/graphql-protect/internal/business/gql"
	"github.com/stretchr/testify/assert"
)

type testLogHandler struct {
	assert func(ctx context.Context, record slog.Record) error
	count  int64 // Use atomic for thread safety in async tests
}

func (t *testLogHandler) Enabled(context.Context, slog.Level) bool {
	return true
}
func (t *testLogHandler) Handle(ctx context.Context, record slog.Record) error {
	atomic.AddInt64(&t.count, 1)
	return t.assert(ctx, record)
}
func (t *testLogHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return t
}
func (t *testLogHandler) WithGroup(_ string) slog.Handler {
	return t
}

func TestAccessLogging_Log(t *testing.T) {
	type args struct {
		cfg      Config
		payloads []gql.RequestData
		headers  http.Header
		count    int64
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
					Enabled:              true,
					IncludedHeaders:      []string{"Authorization", "not-case-sensitive"},
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
					"Authorization":      {"bearer hello"},
					"Content-Type":       {"application/json"},
					"Not-Case-Sensitive": {"yes"},
				},
				count: 1,
			},
			want: func(_ context.Context, record slog.Record) error {
				assert.Equal(t, 1, record.NumAttrs())
				record.Attrs(func(a slog.Attr) bool {
					assert.Equal(t, "payload", a.Key)

					al := a.Value.Any().(accessLog)

					assert.Equal(t, "Foobar", al.OperationName)
					assert.Equal(t, "query Foo { id name }", al.Payload)
					assert.Equal(t, map[string]interface{}{
						"foo": "bar",
					}, al.Variables)
					assert.Equal(t, map[string]interface{}{
						"Authorization":      []string{"bearer hello"},
						"not-case-sensitive": []string{"yes"},
					}, al.Headers)

					return true
				})

				return nil
			},
		},
		{
			name: "logs nothing when disabled",
			args: args{
				cfg: Config{
					Enabled:              false,
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
			want: func(_ context.Context, _ slog.Record) error {
				assert.Fail(t, "should never reach here")
				return nil
			},
		},
	}
	for _, tt := range tests {
		// Test both sync and async modes
		t.Run(tt.name+" (sync)", func(t *testing.T) {
			runAccessLoggingTest(t, tt.args, tt.want, false)
		})
		t.Run(tt.name+" (async)", func(t *testing.T) {
			runAccessLoggingTest(t, tt.args, tt.want, true)
		})
	}
}

func runAccessLoggingTest(t *testing.T, args struct {
	cfg      Config
	payloads []gql.RequestData
	headers  http.Header
	count    int64
}, want func(ctx context.Context, record slog.Record) error, async bool) {
	handler := &testLogHandler{assert: want}
	log := slog.New(handler)

	// Set async mode based on parameter
	cfg := args.cfg
	cfg.Async = async
	cfg.BufferSize = 100 // Small buffer for testing

	a := NewAccessLogging(cfg, log)
	defer func() {
		if async {
			// Gracefully shutdown async logger
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			_ = a.Shutdown(ctx)
		}
	}()

	a.Log(args.payloads, args.headers)

	if async {
		// For async mode, wait a bit for the background goroutine to process
		time.Sleep(50 * time.Millisecond)
	}

	assert.Equal(t, args.count, atomic.LoadInt64(&handler.count))
}
