package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func Timing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)

		duration := time.Since(start).String()
		slog.InfoContext(
			r.Context(),
			"handled request",
			slog.String("duration", duration),
		)
	})
}
