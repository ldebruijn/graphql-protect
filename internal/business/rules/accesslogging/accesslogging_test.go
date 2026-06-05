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

	a, err := NewAccessLogging(cfg, log)
	assert.NoError(t, err)
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

func TestAccessLogging_Shutdown(t *testing.T) {
	handler := &testLogHandler{
		assert: func(_ context.Context, _ slog.Record) error {
			return nil
		},
	}
	log := slog.New(handler)

	t.Run("shutdown with disabled logging", func(t *testing.T) {
		cfg := Config{
			Enabled: false,
		}
		a, err := NewAccessLogging(cfg, log)
		assert.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err = a.Shutdown(ctx)
		assert.NoError(t, err)
	})

	t.Run("shutdown with sync stdout logging", func(t *testing.T) {
		cfg := Config{
			Enabled: true,
			Async:   false,
		}
		a, err := NewAccessLogging(cfg, log)
		assert.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err = a.Shutdown(ctx)
		assert.NoError(t, err)
	})

	t.Run("shutdown with async stdout logging", func(t *testing.T) {
		cfg := Config{
			Enabled:    true,
			Async:      true,
			BufferSize: 100,
		}
		a, err := NewAccessLogging(cfg, log)
		assert.NoError(t, err)

		// Log some entries
		a.Log([]gql.RequestData{
			{OperationName: "TestOp"},
		}, http.Header{})

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err = a.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

// mockLogWriter counts Write invocations so regression tests can assert
// delivery without contacting an external backend.
type mockLogWriter struct {
	writes int
}

func (m *mockLogWriter) Write(_ LogEntryData)             { m.writes++ }
func (m *mockLogWriter) Shutdown(_ context.Context) error { return nil }

// Regression: enabling both Async and Google Cloud Logging used to silently
// drop every log entry. The async buffer is intentionally not created when
// GCP is the backend (the GCP client batches internally), but the `async`
// field was left true — so Log() routed into the async branch, hit a nil
// channel, and fell through to the dropped_logs default case. After the fix,
// `async` is effectively disabled when GCP is enabled and logs flow
// synchronously to the writer.
func TestAccessLogging_GCPWithAsyncDoesNotDropLogs(t *testing.T) {
	cfg := Config{
		Enabled:    true,
		Async:      true,
		BufferSize: 100,
		GoogleCloudLogging: GoogleCloudConfig{
			Enabled: true,
			// Empty ProjectID makes NewGoogleCloudWriter fail before any
			// network IO, keeping the test hermetic. We swap the writer below.
		},
	}
	log := slog.New(&testLogHandler{assert: func(_ context.Context, _ slog.Record) error { return nil }})

	a, err := NewAccessLogging(cfg, log)
	assert.NoError(t, err)
	assert.False(t, a.async, "async must be disabled when GCP is enabled")
	assert.Nil(t, a.logChan, "async channel must not be created when GCP is enabled")

	mock := &mockLogWriter{}
	a.writer = mock

	a.Log([]gql.RequestData{{OperationName: "Test"}}, http.Header{})

	assert.Equal(t, 1, mock.writes, "log must reach the writer instead of being dropped")
}

// Regression: when GCP writer construction fails, the constructor must fall
// back to the stdout writer as the warning message promises. Previously the
// initial `writer = NewStdoutWriter(log)` assignment was clobbered by the
// failed `NewGoogleCloudWriter` call (which returns a nil pointer alongside
// the error), leaving the writer field holding an interface that wraps a nil
// concrete pointer — every subsequent Log() call would NPE.
func TestAccessLogging_FallsBackToStdoutWhenGCPInitFails(t *testing.T) {
	cfg := Config{
		Enabled: true,
		GoogleCloudLogging: GoogleCloudConfig{
			Enabled: true,
			// Empty ProjectID makes NewGoogleCloudWriter return an error before
			// any network IO, triggering the fallback path.
		},
	}
	log := slog.New(&testLogHandler{assert: func(_ context.Context, _ slog.Record) error { return nil }})

	a, err := NewAccessLogging(cfg, log)
	assert.NoError(t, err)
	assert.IsType(t, &StdoutWriter{}, a.writer, "must fall back to stdout writer when GCP init fails")

	// Calling Log() must not panic on the nil GCP writer.
	assert.NotPanics(t, func() {
		a.Log([]gql.RequestData{{OperationName: "Test"}}, http.Header{})
	})
}
