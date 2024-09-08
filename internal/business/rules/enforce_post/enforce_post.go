package enforce_post // nolint:revive

import (
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
)

var methodCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "graphql_protect",
	Subsystem: "enforce_post",
	Name:      "count",
	Help:      "Amount of times the enforce POST rule was triggered and blocked a request",
},
	[]string{},
)

func init() {
	prometheus.MustRegister(methodCounter)
}

type Config struct {
	Enabled bool `yaml:"enabled"`
}

func DefaultConfig() Config {
	return Config{
		Enabled: true,
	}
}

func EnforcePostMethod(cfg Config) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if !cfg.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			query := r.URL.Query()

			if r.Method != "POST" && (query.Has("query") || query.Has("extensions")) {
				methodCounter.WithLabelValues().Inc()
				http.Error(w, "405 - method not allowed", http.StatusMethodNotAllowed)
				return
			}

			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
