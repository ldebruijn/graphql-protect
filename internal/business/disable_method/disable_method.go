package disable_method

import (
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
)

var methodCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "go_graphql_armor",
	Subsystem: "disable_get_method",
	Name:      "count",
	Help:      "Amount of times the disable method rule was triggered",
},
	[]string{},
)

func init() {
	prometheus.MustRegister(methodCounter)
}

type Config struct {
	Enabled bool `conf:"default:true" yaml:"enabled"`
}

func DisableMethodRule(cfg Config) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if !cfg.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			if r.Method == "GET" {
				methodCounter.WithLabelValues().Inc()
				http.Error(w, "405 - method not allowed", http.StatusMethodNotAllowed)
				return
			}

			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
