package main

import (
	"context"
	"log/slog"
)

type CustomHandler struct {
	slog.Handler
}

func (l CustomHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx.Value("requestId") == nil {
		return l.Handler.Handle(ctx, r)
	}

	requestId := ctx.Value("requestId").(string)
	sourceIp := ctx.Value("sourceIp").(string)
	path := ctx.Value("path").(string)
	method := ctx.Value("method").(string)
	userId := ctx.Value("userId")

	requestGroup := slog.Group(
		"request",
		slog.String("requestId", requestId),
		slog.String("sourceIp", sourceIp),
		slog.String("method", method),
		slog.String("path", path),
		slog.Any("userId", userId),
	)

	r.AddAttrs(requestGroup)

	return l.Handler.Handle(ctx, r)
}
