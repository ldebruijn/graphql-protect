package protect

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	validationDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "graphql_protect",
		Subsystem: "validation",
		Name:      "duration_seconds",
		Help:      "Duration of validation phases, excluding upstream latency, in seconds",
		Buckets:   []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.5, 1.0},
	},
		[]string{"phase", "result"},
	)

	upstreamDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "graphql_protect",
		Subsystem: "upstream",
		Name:      "duration_seconds",
		Help:      "Duration of upstream GraphQL server requests in seconds",
		Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
	},
		[]string{"route"},
	)
)

func init() {
	prometheus.MustRegister(validationDuration)
	prometheus.MustRegister(upstreamDuration)
}

type timingContextKey struct{}

// TimingContext tracks request timing through the validation pipeline
type TimingContext struct {
	Start      time.Time
	Phases     map[string]time.Duration
	ProtectEnd time.Time
}

// NewTimingContext creates a new timing context
func NewTimingContext() *TimingContext {
	return &TimingContext{
		Start:  time.Now(),
		Phases: make(map[string]time.Duration),
	}
}

// RecordPhase records the duration of a validation phase
func (tc *TimingContext) RecordPhase(phase string, duration time.Duration) {
	tc.Phases[phase] = duration
}

// End marks the end of protect validation (before proxying upstream)
func (tc *TimingContext) End() {
	tc.ProtectEnd = time.Now()
}

// Duration returns the total duration spent in protect validation
func (tc *TimingContext) Duration() time.Duration {
	if tc.ProtectEnd.IsZero() {
		return 0
	}
	return tc.ProtectEnd.Sub(tc.Start)
}

// WithTimingContext adds a TimingContext to the request context
func WithTimingContext(ctx context.Context, tc *TimingContext) context.Context {
	return context.WithValue(ctx, timingContextKey{}, tc)
}

// TimingContextFromContext retrieves the TimingContext from the request context
func TimingContextFromContext(ctx context.Context) *TimingContext {
	tc, ok := ctx.Value(timingContextKey{}).(*TimingContext)
	if !ok {
		return nil
	}
	return tc
}

// RecordValidationDuration records a validation phase duration to Prometheus
func RecordValidationDuration(phase, result string, duration time.Duration) {
	validationDuration.WithLabelValues(phase, result).Observe(duration.Seconds())
}

// RecordUpstreamDuration records the upstream duration to Prometheus
func RecordUpstreamDuration(route string, duration time.Duration) {
	upstreamDuration.WithLabelValues(route).Observe(duration.Seconds())
}
