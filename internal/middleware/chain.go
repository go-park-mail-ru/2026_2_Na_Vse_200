// Package middleware содержит обвязку вокруг HTTP-обработчиков: идентификатор
// запроса, восстановление после паники, логирование, CORS и подмену текстовых
// ошибок маршрутизатора.
//
// Вынесено из handlers сознательно: обвязка не знает ни про пользователей, ни
// про каталог — она одинаково работает с любым запросом. У неё своя зона
// ответственности, поэтому и свой пакет.
package middleware

import (
	"net/http"
)

// Middleware — обёртка вокруг обработчика. Обёртки навешиваются на весь mux сразу,
// а не на каждую ручку по отдельности.
type Middleware func(http.Handler) http.Handler

// Chain навешивает обёртки на handler.
//
// Порядок: первая в списке оказывается самой внешней, то есть получает запрос
// первой и отдаёт ответ последней. В main порядок такой:
// WithRequestID → WithRecover → WithLogging → WithCORS → WithJSONErrors → mux.
//
// WithRequestID идёт первым, чтобы идентификатор был доступен и логу, и записи
// о панике. WithRecover — сразу следом, снаружи остальных: иначе паника
// из соседней обёртки прошла бы мимо него и уронила процесс.
func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// statusWriter запоминает статус ответа: сам http.ResponseWriter его не отдаёт.
//
// Встраивание интерфейса (http.ResponseWriter без имени поля) означает, что все
// методы интерфейса у обёртки уже есть — мы переопределяем только нужный.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Write нужен на случай, когда обработчик пишет тело без явного WriteHeader:
// тогда уходит 200, и в лог должен попасть именно он.
func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}
