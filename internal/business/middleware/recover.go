package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

func Recover(log *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				err := recover()
				// we don't recover http.ErrAbortHandler so the response to the client is aborted, this should not be logged
				if err == http.ErrAbortHandler {
					panic(err)
				}
				if err != nil {
					log.Error("Panic during handling of request", "error", err, "method", r.Method, "path", r.URL.Path, "stacktrace", string(debug.Stack()))
				}
			}()

			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
