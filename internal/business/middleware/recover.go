package middleware

import (
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"net/http"
	"runtime/debug"
)

var recoverCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "go_graphql_armor",
	Subsystem: "recover",
	Name:      "count",
	Help:      "Amount of times the middleware recovered a panic",
},
	[]string{"error"},
)

func init() {
	prometheus.MustRegister(recoverCounter)
}

func Recover(log *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				err := recover()
				// we don't recover http.ErrAbortHandler so the response to the client is aborted, this should not be logged
				if err == http.ErrAbortHandler { // nolint:errorlint
					panic(err)
				}
				if err != nil {
					recoverCounter.WithLabelValues(getErrNameFromAny(err)).Inc()
					log.Error("Panic during handling of request", "error", err, "method", r.Method, "path", r.URL.Path, "stacktrace", string(debug.Stack()))
				}
			}()

			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

func getErrNameFromAny(err any) string {
	switch val := err.(type) {
	case error:
		return val.Error()
	default:
		return "unknown"
	}
}
