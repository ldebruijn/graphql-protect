package protect

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNewTimingContext(t *testing.T) {
	tc := NewTimingContext()

	require.NotNil(t, tc)
	assert.NotZero(t, tc.Start)
	assert.NotNil(t, tc.Phases)
	assert.True(t, tc.ProtectEnd.IsZero())
}

func TestTimingContext_RecordPhase(t *testing.T) {
	tc := NewTimingContext()

	tc.RecordPhase("test_phase", 100*time.Millisecond)

	assert.Equal(t, 100*time.Millisecond, tc.Phases["test_phase"])
}

func TestTimingContext_MarkProtectEnd(t *testing.T) {
	tc := NewTimingContext()

	tc.MarkProtectEnd()

	assert.False(t, tc.ProtectEnd.IsZero())
	assert.True(t, tc.ProtectEnd.After(tc.Start))
}

func TestTimingContext_ProtectDuration(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*TimingContext)
		expectedResult func(*TimingContext) bool
	}{
		{
			name: "zero when protect end not marked",
			setupFunc: func(_ *TimingContext) {
				// Don't mark protect end
			},
			expectedResult: func(tc *TimingContext) bool {
				return tc.ProtectDuration() == 0
			},
		},
		{
			name: "positive duration when protect end marked",
			setupFunc: func(tc *TimingContext) {
				time.Sleep(10 * time.Millisecond)
				tc.MarkProtectEnd()
			},
			expectedResult: func(tc *TimingContext) bool {
				duration := tc.ProtectDuration()
				return duration > 0 && duration < 1*time.Second
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := NewTimingContext()
			tt.setupFunc(tc)
			assert.True(t, tt.expectedResult(tc))
		})
	}
}

func TestWithTimingContext_AndRetrieve(t *testing.T) {
	tc := NewTimingContext()
	ctx := context.Background()

	// Add to context
	newCtx := WithTimingContext(ctx, tc)

	// Retrieve from context
	retrieved := TimingContextFromContext(newCtx)

	require.NotNil(t, retrieved)
	assert.Equal(t, tc, retrieved)
}

func TestTimingContextFromContext_Nil(t *testing.T) {
	ctx := context.Background()

	retrieved := TimingContextFromContext(ctx)

	assert.Nil(t, retrieved)
}

func TestRecordValidationDuration(_ *testing.T) {
	// This is a smoke test - just verify it doesn't panic
	RecordValidationDuration("test_phase", "success", 100*time.Millisecond)
	RecordValidationDuration("test_phase", "error", 50*time.Millisecond)
}

func TestRecordUpstreamDuration(_ *testing.T) {
	// This is a smoke test - just verify it doesn't panic
	RecordUpstreamDuration("/graphql", 100*time.Millisecond)
}

func TestResultFromError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "success when no error",
			err:      nil,
			expected: "success",
		},
		{
			name:     "error when error present",
			err:      assert.AnError,
			expected: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resultFromError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
