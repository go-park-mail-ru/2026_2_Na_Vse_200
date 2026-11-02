package middleware

import (
	"context"
	"net/http"
	"uuid"
)

type ctxKey int

const (
	// HeaderRequestID задаёт имя заголовка с идентификатором запроса.
	HeaderRequestID = "X-Request-ID"

	_requestIDKey ctxKey = 0
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

		ctx := context.WithValue(r.Context(), _requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFromContext возвращает ID запроса или пустую строку.
func RequestIDFromContext(ctx context.Context) string {
	id, ok := ctx.Value(_requestIDKey).(string)
	if !ok {
		return ""
	}
	return id
}
