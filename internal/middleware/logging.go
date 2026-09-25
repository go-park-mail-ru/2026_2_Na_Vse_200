package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

// WithLogging пишет по строке на каждый обработанный запрос.
//
// Лог структурированный (log/slog): не текст, а набор пар «ключ-значение».
// Такую строку можно фильтровать и агрегировать — например, найти все запросы
// с status 500 за час или посчитать среднее duration_ms по ручке.
//
// component отвечает на вопрос «кто обработал»: сейчас это монолит, а когда
// появятся отдельные сервисы, в поле попадёт имя конкретного.
func WithLogging(logger *slog.Logger, component string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w}

			next.ServeHTTP(sw, r)

			duration := time.Since(start)
			if sw.status == 0 {
				// Обработчик не вызвал WriteHeader — значит ушёл 200.
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

// RealIP пытается определить адрес настоящего клиента.
//
// Когда сервис стоит за прокси или балансировщиком, в RemoteAddr оказывается
// адрес прокси, а исходный клиент указывается в заголовках.
//
// Важно: эти заголовки клиент может выставить сам, поэтому доверять им можно
// только тогда, когда сервис действительно спрятан за доверенным прокси,
// который их перезаписывает. Для логов этого достаточно, для проверок
// безопасности — нет.
func RealIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}

	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// Заголовок накапливает цепочку адресов: клиент, потом прокси.
		// Исходный клиент — первый.
		client, _, _ := strings.Cut(forwarded, ",")
		if client = strings.TrimSpace(client); client != "" {
			return client
		}
	}

	// RemoteAddr приходит в виде "адрес:порт" — порт в логе не нужен.
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
