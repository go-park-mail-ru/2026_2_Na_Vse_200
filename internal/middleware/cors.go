package middleware

import (
	"net/http"
	"strings"
)

// WithCORS разрешает браузерные запросы с адреса фронтенда.
//
// Разрешённый адрес ровно один: у сервиса один сайт, и список адресов был бы
// заделом на будущее, за который пришлось бы платить сейчас. Ответ, который
// зависит от заголовка Origin, кэшируется промежуточными прокси, и на таком
// поведении строится отравление кэша: злоумышленник добивается, чтобы в кэш
// попал ответ с чужим разрешением, и раздаёт его остальным. С единственным
// адресом ответ одинаков для всех, и вопрос снимается.
//
// Пустой allowedOrigin означает, что фронтенд и API живут на одном origin
// и CORS не нужен вовсе.
//
// Звёздочка (Allow-Origin: *) несовместима с Allow-Credentials: true — браузер
// в этом случае не отправит cookie, и авторизация работать не будет. Поэтому
// в заголовок пишется конкретный адрес.
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

			// Перед «непростым» запросом браузер шлёт preflight OPTIONS.
			// Если на него не ответить, основной запрос просто не уйдёт.
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
