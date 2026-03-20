package protect

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"sync"
	"time"
)

var (
	validationDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "graphql_protect",
		Subsystem: "validation",
		Name:      "duration_seconds",
		Help:      "Duration of validation phases in seconds",
		Buckets:   []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.5, 1.0},
	},
		[]string{"phase", "result"},
	)

	overheadRatio = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "graphql_protect",
		Subsystem: "protect",
		Name:      "overhead_ratio",
		Help:      "Ratio of protect processing time to total request time (0.0-1.0)",
	},
		[]string{"route"},
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
	prometheus.MustRegister(overheadRatio)
	prometheus.MustRegister(upstreamDuration)
}

type timingContextKey struct{}

// TimingContext tracks request timing through the validation pipeline
type TimingContext struct {
	Start      time.Time
	Phases     map[string]time.Duration
	ProtectEnd time.Time
	mu         sync.RWMutex
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
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.Phases[phase] = duration
}

// MarkProtectEnd marks the end of protect validation (before proxying upstream)
func (tc *TimingContext) MarkProtectEnd() {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.ProtectEnd = time.Now()
}

// ProtectDuration returns the total duration spent in protect validation
func (tc *TimingContext) ProtectDuration() time.Duration {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	if tc.ProtectEnd.IsZero() {
		return 0
	}
	return tc.ProtectEnd.Sub(tc.Start)
}

// OverheadRatio calculates the ratio of protect time to total time (0.0 to 1.0)
func (tc *TimingContext) OverheadRatio(totalDuration time.Duration) float64 {
	if totalDuration == 0 {
		return 0.0
	}
	protectDuration := tc.ProtectDuration()
	if protectDuration == 0 {
		return 0.0
	}
	return protectDuration.Seconds() / totalDuration.Seconds()
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

// RecordOverheadRatio records the overhead ratio to Prometheus
func RecordOverheadRatio(route string, ratio float64) {
	overheadRatio.WithLabelValues(route).Set(ratio)
}

// RecordUpstreamDuration records the upstream duration to Prometheus
func RecordUpstreamDuration(route string, duration time.Duration) {
	upstreamDuration.WithLabelValues(route).Observe(duration.Seconds())
}
