package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/pkg/apimessage"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/pkg/response"
)

// WithRecover ловит панику в обработчике: сервис отвечает 500 в формате
// контракта и продолжает работать, вместо того чтобы уронить весь процесс.
//
// Каждый запрос обрабатывается в своей горутине, а паника в горутине
// без recover завершает всю программу.
//
// В лог попадают и стек, и идентификатор запроса — по нему авария связывается
// с остальными записями того же запроса. Клиенту не уходит ничего из этого:
// текст паники и пути файлов ему знать незачем.
func WithRecover(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.LogAttrs(r.Context(), slog.LevelError, "паника при обработке запроса",
						slog.String("request_id", RequestIDFromContext(r.Context())),
						slog.String("method", r.Method),
						slog.String("url", r.URL.RequestURI()),
						slog.Any("panic", rec),
						slog.String("stack", string(debug.Stack())),
					)

					response.WriteError(w, http.StatusInternalServerError,
						apimessage.CodeInternal, apimessage.MsgInternal)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
