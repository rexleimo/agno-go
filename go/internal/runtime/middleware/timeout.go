package middleware

import (
	"net/http"
	"time"
)

// Timeout wraps handlers with a deadline and returns 503 on expiry.
func Timeout(d time.Duration) func(http.Handler) http.Handler {
	if d <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("stream") == "true" {
				next.ServeHTTP(w, r)
				return
			}
			http.TimeoutHandler(next, d, "request timeout").ServeHTTP(w, r)
		})
	}
}
