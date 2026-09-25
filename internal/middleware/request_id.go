package middleware

import (
	"context"
	"net/http"
	"uuid"
)

// HeaderRequestID — заголовок с идентификатором запроса.
const HeaderRequestID = "X-Request-ID"

// ctxKey — неэкспортируемый тип ключа: строковый ключ мог бы совпасть
// с ключом другого пакета.
type ctxKey int

const requestIDKey ctxKey = iota

// WithRequestID кладёт идентификатор запроса в контекст и в заголовок ответа.
// Присланный клиентом идентификатор переиспользуется, если это валидный UUID.
func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(HeaderRequestID)
		if _, err := uuid.Parse(id); err != nil {
			id = uuid.New().String()
		}

		w.Header().Set(HeaderRequestID, id)

		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFromContext возвращает идентификатор запроса или пустую строку.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}
