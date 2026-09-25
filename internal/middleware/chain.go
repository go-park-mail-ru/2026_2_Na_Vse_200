// Package middleware содержит обвязку вокруг HTTP-обработчиков: идентификатор
// запроса, восстановление после паники, логирование, CORS и подмену текстовых
// ошибок маршрутизатора.
package middleware

import (
	"net/http"
)

// Middleware — обёртка вокруг обработчика.
type Middleware func(http.Handler) http.Handler

// Chain навешивает обёртки на handler. Первая в списке становится самой внешней.
func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// statusWriter запоминает статус ответа: сам http.ResponseWriter его не отдаёт.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Write покрывает случай, когда обработчик пишет тело без явного WriteHeader.
func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}
