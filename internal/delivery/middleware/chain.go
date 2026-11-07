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

// statusWriter сохраняет HTTP-статус и число записанных байтов тела ответа.
type statusWriter struct {
	http.ResponseWriter
	status       int
	bytesWritten int64
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status != 0 {
		return
	}
	// Информационный ответ не фиксирует окончательный статус.
	if code >= 100 && code < 200 && code != http.StatusSwitchingProtocols {
		w.ResponseWriter.WriteHeader(code)
		return
	}
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Write учитывает неявный статус 200 при записи тела ответа.
func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += int64(n)
	return n, err
}

// Unwrap позволяет http.ResponseController получить исходный ResponseWriter.
func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
