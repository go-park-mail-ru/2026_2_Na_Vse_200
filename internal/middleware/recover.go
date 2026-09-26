package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/apimessage"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/pkg/response"
)

// WithRecover перехватывает панику: клиент получает 500 в формате контракта,
// стек и идентификатор запроса остаются в логе.
//
// Идентификатор берётся из заголовка ответа, а не из контекста: обёртка стоит
// снаружи всех остальных, и контекст с идентификатором сюда не доходит.
func WithRecover(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.LogAttrs(r.Context(), slog.LevelError, "паника при обработке запроса",
						slog.String("request_id", w.Header().Get(HeaderRequestID)),
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
