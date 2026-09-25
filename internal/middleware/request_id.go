package middleware

import (
	"context"
	"net/http"
	"uuid"
)

// HeaderRequestID — заголовок, в котором идентификатор запроса приходит
// и уходит обратно клиенту.
const HeaderRequestID = "X-Request-ID"

// ctxKey — собственный тип для ключей контекста.
//
// Обычная строка в роли ключа опасна: любой другой пакет может случайно
// использовать такую же и перезаписать значение. Неэкспортируемый тип
// гарантирует, что ключ уникален для этого пакета.
type ctxKey int

const requestIDKey ctxKey = iota

// WithRequestID присваивает каждому запросу уникальный идентификатор.
//
// Он попадает в контекст (оттуда его берут логгер и обработчики), в заголовок
// ответа и в каждую строку лога. Благодаря этому по жалобе пользователя
// «у меня ошибка, вот номер» находятся все записи именно его запроса.
//
// Если клиент прислал свой идентификатор — переиспользуем: так цепочка
// вызовов между сервисами связывается в одну историю. Значение проверяется
// на формат UUID, чтобы в логи не попало произвольное содержимое от клиента.
func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(HeaderRequestID)
		if _, err := uuid.Parse(id); err != nil {
			id = uuid.New().String()
		}

		w.Header().Set(HeaderRequestID, id)

		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFromContext возвращает идентификатор запроса.
// Пустая строка означает, что обёртка WithRequestID не была подключена.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}
