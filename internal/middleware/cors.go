package middleware

import (
	"net/http"
	"strings"
)

// WithCORS разрешает браузерные запросы с адреса фронтенда.
// Адрес ровно один: ответ не зависит от заголовка Origin и не открывает дорогу
// к отравлению кэша на прокси. Пустое значение отключает обёртку.
//
// В Allow-Origin идёт конкретный адрес: звёздочка несовместима
// с Allow-Credentials, и cookie ходить не будут.
func WithCORS(allowedOrigin string) Middleware {
	return func(next http.Handler) http.Handler {
		if allowedOrigin == "" {
			return next
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.EqualFold(r.Header.Get("Origin"), allowedOrigin) {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			}

			// Preflight-запрос браузера перед основным.
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
