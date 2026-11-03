package middleware

import (
	"net/http"
)

type Middleware func(http.Handler) http.Handler

// Chain оборачивает обработчик middleware в порядке перечисления: первая обёртка внешняя.
func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// statusWriter сохраняет HTTP-статус для записи в лог.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Write учитывает неявный статус 200 при записи тела ответа.
func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}
