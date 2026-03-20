package middleware

import (
	"github.com/ldebruijn/graphql-protect/internal/business/protect"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"time"
)

var (
	httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "graphql_protect",
		Subsystem: "http",
		Name:      "duration_seconds",
		Help:      "HTTP request duration in seconds, broken down by component",
		Buckets:   prometheus.DefBuckets,
	},
		[]string{"route", "component"},
	)
)

func init() {
	prometheus.MustRegister(httpDuration)
}

func RequestMetricMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			next.ServeHTTP(w, r)

			totalDuration := time.Since(start)
			route := r.URL.Path

			// Record total duration
			httpDuration.WithLabelValues(route, "total").Observe(totalDuration.Seconds())

			// Extract timing context to calculate protect vs upstream
			tc := protect.TimingContextFromContext(r.Context())
			if tc != nil && !tc.ProtectEnd.IsZero() {
				protectDuration := tc.ProtectDuration()
				upstreamDuration := totalDuration - protectDuration

				// Record component durations
				httpDuration.WithLabelValues(route, "protect").Observe(protectDuration.Seconds())
				httpDuration.WithLabelValues(route, "upstream").Observe(upstreamDuration.Seconds())

				// Record overhead ratio and upstream duration
				protect.RecordOverheadRatio(route, tc.OverheadRatio(totalDuration))
				protect.RecordUpstreamDuration(route, upstreamDuration)
			}
		}
		return http.HandlerFunc(fn)
	}
}
