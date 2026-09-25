package middleware

import (
	"net/http"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/pkg/apimessage"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/pkg/response"
)

// WithJSONErrors подменяет текстовые ответы ServeMux на JSON нашего формата.
//
// Неизвестный путь и неподходящий метод обрабатывает сам ServeMux, и отвечает он
// строкой вроде "404 page not found" с Content-Type text/plain. Контракт требует
// JSON, а перехватить эти ответы можно только обёрткой над ResponseWriter.
func WithJSONErrors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&errorInterceptor{ResponseWriter: w}, r)
	})
}

// errorInterceptor перехватывает статусы 404 и 405 и пишет вместо них наш JSON.
// Заголовок Allow у 405 при этом сохраняется: ServeMux ставит его до WriteHeader.
type errorInterceptor struct {
	http.ResponseWriter
	replaced bool
}

func (w *errorInterceptor) WriteHeader(code int) {
	switch code {
	case http.StatusNotFound:
		w.replaced = true
		response.WriteError(w.ResponseWriter, code, apimessage.CodeNotFound, apimessage.MsgNotFound)
	case http.StatusMethodNotAllowed:
		w.replaced = true
		response.WriteError(w.ResponseWriter, code, apimessage.CodeMethodNotAllowed, apimessage.MsgMethodNotAllowed)
	default:
		w.ResponseWriter.WriteHeader(code)
	}
}

// Write выбрасывает текст, который ServeMux пишет после своего WriteHeader:
// тело мы уже отправили сами. Клиенту сообщаем, что запись «удалась», иначе
// стандартная библиотека сочтёт это ошибкой соединения.
func (w *errorInterceptor) Write(b []byte) (int, error) {
	if w.replaced {
		return len(b), nil
	}
	return w.ResponseWriter.Write(b)
}
