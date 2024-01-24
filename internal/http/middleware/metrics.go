package middleware

import (
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"time"
)

var (
	httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "graphql_protect",
		Subsystem: "http",
		Name:      "duration",
		Help:      "HTTP duration",
	},
		[]string{"route"},
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

			httpDuration.WithLabelValues(r.URL.Path).Observe(time.Since(start).Seconds())
		}
		return http.HandlerFunc(fn)
	}
}
