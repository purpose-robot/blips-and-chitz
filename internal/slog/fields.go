package slog

import (
	"context"
	"log/slog"
	"slices"
	"sync"
)

type logFields struct {
	mu     sync.Mutex
	fields []slog.Attr
}

type contextKey struct{}

func Fields(ctx context.Context) []slog.Attr {
	fields, ok := ctx.Value(contextKey{}).(*logFields)
	if !ok {
		return nil
	}

	fields.mu.Lock()
	defer fields.mu.Unlock()

	return slices.Clone(fields.fields)
}

func AddField(ctx context.Context, attrs ...slog.Attr) {
	lf, ok := ctx.Value(contextKey{}).(*logFields)
	if ok {
		lf.mu.Lock()
		lf.fields = append(lf.fields, attrs...)
		lf.mu.Unlock()
	}
}

func WithFields(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKey{}, &logFields{})
}
