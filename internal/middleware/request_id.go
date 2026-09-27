package middleware

import (
	"context"
	"net/http"
	"uuid"
)

// HeaderRequestID — имя заголовка, в котором идентификатор приходит и уходит.
const HeaderRequestID = "X-Request-ID"

type ctxKey int

const _requestIDKey ctxKey = iota

// WithRequestID кладёт идентификатор запроса в контекст и в заголовок ответа.
// Присланный клиентом идентификатор переиспользуется, если это валидный UUID.
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

// RequestIDFromContext возвращает идентификатор запроса из ctx.
func RequestIDFromContext(ctx context.Context) string {
	id, ok := ctx.Value(_requestIDKey).(string)
	if !ok {
		return ""
	}
	return id
}
