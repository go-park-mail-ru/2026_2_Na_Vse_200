package middleware

import (
	"log/slog"
	"net/http"
	"runtime"

	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
)

const _stackBufferSize = 4 << 10

// WithRecover перехватывает панику и возвращает ответ 500.
// ID запроса берётся из заголовка ответа: контекст внутренних обёрток здесь недоступен.
func WithRecover(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					// Размер стека ограничен, чтобы не записывать в лог слишком большой объём данных.
					stackTrace := make([]byte, _stackBufferSize)
					stackTrace = stackTrace[:runtime.Stack(stackTrace, true)]

					logger.LogAttrs(r.Context(), slog.LevelError, "паника при обработке запроса",
						slog.String("request_id", w.Header().Get(HeaderRequestID)),
						slog.String("method", r.Method),
						slog.String("url", r.URL.RequestURI()),
						slog.Any("panic", rec),
						slog.String("stack", string(stackTrace)),
					)

					response.WriteError(w, http.StatusInternalServerError,
						models.CodeInternal, models.MsgInternal)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
