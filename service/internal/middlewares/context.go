package middlewares

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func SetContext() fiber.Handler {
	return func(c fiber.Ctx) error {
		slog.Debug("adding additional request context")
		requestId := uuid.NewString()
		ctx := c.Context()

		ctx = context.WithValue(ctx, "requestId", requestId)
		ctx = context.WithValue(ctx, "method", c.Method())
		ctx = context.WithValue(ctx, "path", c.Path())
		ctx = context.WithValue(ctx, "sourceIp", c.IP())

		c.SetContext(ctx)
		c.Response().Header.Set("X-Request-ID", requestId)

		return c.Next()
	}
}
