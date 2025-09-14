package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

func SetContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Debug("adding additional request context")

		requestID := uuid.NewString()

		ctx := r.Context()
		ctx = context.WithValue(ctx, "requestId", requestID)
		ctx = context.WithValue(ctx, "method", r.Method)
		ctx = context.WithValue(ctx, "path", r.URL.Path)
		ctx = context.WithValue(ctx, "sourceIp", r.RemoteAddr)

		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
