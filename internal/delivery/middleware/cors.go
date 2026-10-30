package middleware

import (
	"net/http"
	"strings"
)

// WithCORS разрешает браузерные запросы с адреса фронтенда.
// Пустой origin отключает обёртку. Звёздочка в Allow-Origin несовместима
// с Allow-Credentials, поэтому адрес указывается точный.
func WithCORS(allowedOrigin, allowedMethods, allowedHeaders string) Middleware {
	return func(next http.Handler) http.Handler {
		if allowedOrigin == "" {
			return next
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.EqualFold(r.Header.Get("Origin"), allowedOrigin) {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
				w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
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
