package middleware

import (
	"context"
	"net/http"
	"uuid"

	"2026_2_Na_Vse_200/internal/logger"
)

const (
	// HeaderRequestID задаёт имя заголовка с идентификатором запроса.
	HeaderRequestID = "X-Request-ID"
)

// WithRequestID сохраняет ID запроса в контексте и заголовке ответа.
// ID клиента используется только при корректном формате UUID.
func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(HeaderRequestID)
		if _, err := uuid.Parse(id); err != nil {
			id = uuid.New().String()
		}

		w.Header().Set(HeaderRequestID, id)

		ctx := logger.WithRequestID(r.Context(), id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFromContext возвращает ID запроса или пустую строку.
func RequestIDFromContext(ctx context.Context) string {
	return logger.RequestID(ctx)
}
