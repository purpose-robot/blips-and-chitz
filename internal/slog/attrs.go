package slog

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

func Error(err error) slog.Attr {
	return slog.String("error", err.Error())
}

func Cause(err error) slog.Attr {
	return slog.String("cause", err.Error())
}

func Address(address int) slog.Attr {
	return slog.String("address", fmt.Sprintf(":%d", address))
}

func Request(r *http.Request, status, bytesCount int, start time.Time) slog.Attr {
	return slog.Group(
		"request",
		slog.String("path", r.URL.Path),
		slog.String("method", r.Method),
		slog.Int("status_code", status),
		slog.Int("bytes_count", bytesCount),
		slog.Duration("duration", time.Since(start)),
	)
}
