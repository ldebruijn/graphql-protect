package accesslogging

import (
	"context"
	"log/slog"
	"sync/atomic"
	"testing"

	"github.com/ldebruijn/graphql-protect/internal/business/gql"
	"github.com/stretchr/testify/assert"
)

func TestStdoutWriter_Write(t *testing.T) {
	tests := []struct {
		name string
		data LogEntryData
		want func(ctx context.Context, record slog.Record) error
	}{
		{
			name: "writes all fields when configured",
			data: LogEntryData{
				Payloads: []gql.RequestData{
					{
						OperationName: "TestOp",
						Variables: map[string]interface{}{
							"id": "123",
						},
						Query: "query TestOp { user { id } }",
					},
				},
				FilteredHeaders: map[string]interface{}{
					"Authorization": []string{"Bearer token"},
				},
				IncludeOpName:  true,
				IncludeVars:    true,
				IncludePayload: true,
			},
			want: func(_ context.Context, record slog.Record) error {
				assert.Equal(t, 1, record.NumAttrs())
				record.Attrs(func(a slog.Attr) bool {
					assert.Equal(t, "payload", a.Key)

					al := a.Value.Any().(accessLog)
					assert.Equal(t, "TestOp", al.OperationName)
					assert.Equal(t, "query TestOp { user { id } }", al.Payload)
					assert.Equal(t, map[string]interface{}{"id": "123"}, al.Variables)
					assert.Equal(t, map[string]interface{}{
						"Authorization": []string{"Bearer token"},
					}, al.Headers)

					return true
				})
				return nil
			},
		},
		{
			name: "excludes fields when not configured",
			data: LogEntryData{
				Payloads: []gql.RequestData{
					{
						OperationName: "TestOp",
						Variables:     map[string]interface{}{"id": "123"},
						Query:         "query TestOp { user { id } }",
					},
				},
				FilteredHeaders: map[string]interface{}{},
				IncludeOpName:   false,
				IncludeVars:     false,
				IncludePayload:  false,
			},
			want: func(_ context.Context, record slog.Record) error {
				assert.Equal(t, 1, record.NumAttrs())
				record.Attrs(func(a slog.Attr) bool {
					al := a.Value.Any().(accessLog)
					assert.Empty(t, al.OperationName)
					assert.Empty(t, al.Payload)
					assert.Nil(t, al.Variables)
					assert.Empty(t, al.Headers)
					return true
				})
				return nil
			},
		},
		{
			name: "includes pre-filtered headers",
			data: LogEntryData{
				Payloads: []gql.RequestData{
					{OperationName: "TestOp"},
				},
				FilteredHeaders: map[string]interface{}{
					"Authorization":      []string{"secret"},
					"not-case-sensitive": []string{"included"},
				},
				IncludeOpName:  true,
				IncludeVars:    false,
				IncludePayload: false,
			},
			want: func(_ context.Context, record slog.Record) error {
				record.Attrs(func(a slog.Attr) bool {
					al := a.Value.Any().(accessLog)
					assert.Len(t, al.Headers, 2)
					assert.Contains(t, al.Headers, "Authorization")
					assert.Contains(t, al.Headers, "not-case-sensitive")
					return true
				})
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &testLogHandler{assert: tt.want}
			log := slog.New(handler)

			writer := NewStdoutWriter(log)
			writer.Write(tt.data)

			assert.Equal(t, int64(len(tt.data.Payloads)), atomic.LoadInt64(&handler.count))
		})
	}
}

func TestStdoutWriter_Shutdown(t *testing.T) {
	log := slog.New(&testLogHandler{})
	writer := NewStdoutWriter(log)

	err := writer.Shutdown(context.Background())
	assert.NoError(t, err, "Shutdown should not return an error")
}
