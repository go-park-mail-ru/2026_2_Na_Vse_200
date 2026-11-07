package logger

import (
	"context"
	"log/slog"
)

type ctxKey struct{}

// WithRequestID кладёт идентификатор запроса в контекст.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// RequestID возвращает идентификатор запроса из контекста или пустую строку.
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

// FromContext возвращает логгер с request_id текущего запроса.
func FromContext(ctx context.Context, base *slog.Logger) *slog.Logger {
	if base == nil {
		base = slog.Default()
	}
	if id := RequestID(ctx); id != "" {
		return base.With(slog.String("request_id", id))
	}
	return base
}

// Discard возвращает логгер без вывода — для тестов.
func Discard() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
