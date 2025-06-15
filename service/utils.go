package main

import (
	"context"
	"log/slog"
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
