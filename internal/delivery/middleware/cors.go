package middleware

import (
	"net/http"
	"strings"
)

// WithCORS разрешает запросы с указанного origin; пустое значение отключает обёртку.
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

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
