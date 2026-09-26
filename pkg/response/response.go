// Package response пишет HTTP-ответы в формате JSON.
//
// Пакет не знает ни про коды ошибок сервиса, ни про его предметную область:
// коды и тексты передаются вызывающим.
package response

import (
	"encoding/json"
	"log"
	"net/http"
)

// ErrorResponse — внешняя обёртка ошибки: {"error": {...}}.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody — тело ошибки. Fields заполняется только при ошибках валидации.
type ErrorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// WriteJSON сериализует data в JSON и отправляет с кодом состояния code.
// Тело собирается до отправки заголовков: иначе при ошибке сериализации
// клиент получил бы половину ответа после уже отправленного статуса.
// Ошибки записи попадают в лог: клиенту сообщить о них уже нечем.
func WriteJSON(w http.ResponseWriter, code int, data any) {
	body, err := json.Marshal(data)
	if err != nil {
		log.Printf("сборка JSON-ответа: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)

	if _, err := w.Write(body); err != nil {
		log.Printf("отправка ответа: %v", err)
	}
}

// WriteError отправляет ошибку с кодом состояния code. Аргумент errCode —
// машинный код для клиента, message — текст, который можно показать
// пользователю.
func WriteError(w http.ResponseWriter, code int, errCode, message string) {
	WriteJSON(w, code, ErrorResponse{
		Error: ErrorBody{Code: errCode, Message: message},
	})
}

// WriteFieldsError отправляет ошибку так же, как WriteError, и добавляет
// разбор по полям: ключ в fields — имя поля запроса, значение — текст ошибки.
func WriteFieldsError(w http.ResponseWriter, code int, errCode, message string, fields map[string]string) {
	WriteJSON(w, code, ErrorResponse{
		Error: ErrorBody{Code: errCode, Message: message, Fields: fields},
	})
}
