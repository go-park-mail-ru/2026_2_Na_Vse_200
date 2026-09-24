package handlers

import (
	"log"
	"net/http"
	"strings"
	"time"
)

// Middleware — обёртка вокруг обработчика. Обёртки навешиваются на весь mux сразу,
// а не на каждую ручку по отдельности.
type Middleware func(http.Handler) http.Handler

// Wrap навешивает обёртки на handler.
//
// Порядок: первая в списке оказывается самой внешней, то есть получает запрос
// первой и отдаёт ответ последней. В main порядок такой:
// withRecover → withLogging → withCORS → withJSONErrors → mux.
// withRecover обязан быть снаружи всех, иначе паника из другой обёртки пройдёт мимо него.
func Wrap(handler http.Handler, middlewares ...Middleware) http.Handler {
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

// WithRecover ловит панику в обработчике: сервис отвечает 500 в нашем формате
// и продолжает работать, вместо того чтобы уронить весь процесс.
//
// Каждый запрос обрабатывается в своей горутине, а паника в горутине без recover
// завершает всю программу.
func WithRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("паника при обработке %s %s: %v", r.Method, r.URL.Path, rec)
				writeError(w, http.StatusInternalServerError, CodeInternal, "Внутренняя ошибка сервера")
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// WithLogging пишет в лог метод, путь, статус и длительность запроса.
func WithLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w}

		next.ServeHTTP(sw, r)

		if sw.status == 0 {
			sw.status = http.StatusOK
		}
		log.Printf("%s %s -> %d за %s", r.Method, r.URL.Path, sw.status, time.Since(start))
	})
}

// WithCORS разрешает запросы с точно указанных адресов фронтенда.
//
// Звёздочка (Allow-Origin: *) несовместима с Allow-Credentials: true — браузер
// в этом случае не отправит cookie, и авторизация работать не будет. Поэтому
// в заголовок пишется конкретный origin из списка разрешённых.
func WithCORS(allowedOrigins []string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if origin != "" && originAllowed(origin, allowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
				// Ответ зависит от Origin — без этого прокси может отдать чужой кэш.
				w.Header().Add("Vary", "Origin")
			}

			// Перед «непростым» запросом браузер шлёт preflight OPTIONS.
			// Если на него не ответить, основной запрос просто не уйдёт.
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func originAllowed(origin string, allowed []string) bool {
	for _, candidate := range allowed {
		if strings.EqualFold(origin, candidate) {
			return true
		}
	}
	return false
}

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
		writeError(w.ResponseWriter, code, CodeNotFound, "Адрес не найден")
	case http.StatusMethodNotAllowed:
		w.replaced = true
		writeError(w.ResponseWriter, code, CodeMethodNotAllowed, "Метод не поддерживается этим адресом")
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
