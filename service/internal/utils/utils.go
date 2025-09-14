package utils

import (
	"context"
	"crypto/rand"
	"encoding/base64"
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

func GenerateRand(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
