package middleware

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// WithLogging записывает параметры запроса, статус и время обработки.
func WithLogging(logger zerolog.Logger, handledBy string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w}
			requestLogger := logger.With().
				Str("request_id", RequestIDFromContext(r.Context())).Logger()
			r = r.WithContext(requestLogger.WithContext(r.Context()))

			next.ServeHTTP(sw, r)

			duration := time.Since(start)
			if sw.status == 0 {
				sw.status = http.StatusOK
			}

			requestLogger.WithLevel(accessLogLevel(sw.status)).
				Str("method", r.Method).
				Str("url", r.URL.Path).
				Str("host", r.Host).
				Str("remote_addr", r.RemoteAddr).
				Str("real_ip", RealIP(r)).
				Str("user_agent", r.UserAgent()).
				Int64("content_length", r.ContentLength).
				Int64("response_bytes", sw.bytesWritten).
				Int("status", sw.status).
				Time("start_time", start).
				Str("duration_human_readable", duration.String()).
				Float64("duration_ms", float64(duration.Nanoseconds())/1e6).
				Str("handled_by", handledBy).
				Msg("http-запрос обработан")
		})
	}
}

func accessLogLevel(status int) zerolog.Level {
	switch status / 100 {
	case 4:
		return zerolog.WarnLevel
	case 5:
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// RealIP определяет адрес клиента для логов.
// Заголовки прокси могут быть подделаны и не подходят для проверки доступа.
func RealIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}

	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {

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
