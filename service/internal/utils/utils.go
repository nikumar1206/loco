package utils

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v3"
)

func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func LogThrowable(c context.Context, err error) {
	if err != nil {
		slog.ErrorContext(c, err.Error())
	}
}

type Error struct {
	Message   string `json:"message"`
	RequestId string `json:"requestId"`
}

func SendErrorResponse(c fiber.Ctx, statusCode int, message string) error {
	return c.Status(statusCode).JSON(
		Error{
			Message:   message,
			RequestId: c.GetRespHeader("X-Request-ID"),
		},
	)
}