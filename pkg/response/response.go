// Package response пишет HTTP-ответы в формате контракта API.
package response

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/pkg/apimessage"
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

// WriteJSON отправляет data как JSON с указанным статусом.
// Тело собирается до отправки заголовков: иначе при ошибке сериализации
// клиент получил бы половину ответа после уже отправленного статуса.
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

// WriteError отправляет ошибку в формате контракта.
func WriteError(w http.ResponseWriter, code int, errCode, message string) {
	WriteJSON(w, code, ErrorResponse{
		Error: ErrorBody{
			Code:    errCode,
			Message: message,
		},
	})
}

// WriteValidationError отправляет 400 с разбором по полям запроса.
func WriteValidationError(w http.ResponseWriter, fields map[string]string) {
	WriteJSON(w, http.StatusBadRequest, ErrorResponse{
		Error: ErrorBody{
			Code:    apimessage.CodeValidationFailed,
			Message: apimessage.MsgValidationFailed,
			Fields:  fields,
		},
	})
}

// WriteInternalError логирует причину и отдаёт клиенту общий текст без деталей.
func WriteInternalError(w http.ResponseWriter, context string, err error) {
	log.Printf("%s: %v", context, err)
	WriteError(w, http.StatusInternalServerError, apimessage.CodeInternal, apimessage.MsgInternal)
}
