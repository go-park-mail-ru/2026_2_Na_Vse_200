package middleware

import (
	"net/http"
	"runtime"

	"github.com/rs/zerolog"

	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
)

const _stackBufferSize = 4 << 10

// WithRecover перехватывает панику и возвращает ответ 500.
// Вызывается внутри WithLogging, чтобы использовать логгер запроса из контекста.
func WithRecover() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {

					stackTrace := make([]byte, _stackBufferSize)
					stackTrace = stackTrace[:runtime.Stack(stackTrace, false)]

					zerolog.Ctx(r.Context()).Error().
						Str("method", r.Method).
						Str("url", r.URL.Path).
						Interface("panic", rec).
						Str("stack", string(stackTrace)).
						Msg("паника при обработке запроса")

					response.WriteError(r.Context(), w, http.StatusInternalServerError,
						models.CodeInternal, models.MsgInternal)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
