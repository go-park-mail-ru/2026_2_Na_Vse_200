package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

// WithLogging пишет структурированную запись на каждый обработанный запрос.
// component попадает в поле handled_by и отвечает, кто обработал запрос.
func WithLogging(logger *slog.Logger, component string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w}

			next.ServeHTTP(sw, r)

			duration := time.Since(start)
			if sw.status == 0 {
				sw.status = http.StatusOK
			}

			logger.LogAttrs(r.Context(), slog.LevelInfo, "http-запрос обработан",
				slog.String("request_id", RequestIDFromContext(r.Context())),
				slog.String("method", r.Method),
				slog.String("url", r.URL.RequestURI()),
				slog.String("host", r.Host),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("real_ip", RealIP(r)),
				slog.String("user_agent", r.UserAgent()),
				slog.Int64("content_length", r.ContentLength),
				slog.Int("status", sw.status),
				slog.Time("start_time", start),
				slog.String("duration_human_readable", duration.String()),
				slog.Float64("duration_ms", float64(duration.Nanoseconds())/1e6),
				slog.String("handled_by", component),
			)
		})
	}
}

// RealIP определяет адрес клиента с учётом заголовков прокси.
// Заголовки подделываются клиентом, поэтому годятся только для логов.
func RealIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}

	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// В цепочке адресов исходный клиент идёт первым.
		client, _, _ := strings.Cut(forwarded, ",")
		if client = strings.TrimSpace(client); client != "" {
			return client
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
