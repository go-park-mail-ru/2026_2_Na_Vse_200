package response

import (
	"encoding/json"
	"log"
	"net/http"
)

// ErrorResponse содержит ошибку в формате API.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody содержит код, сообщение и необязательные ошибки полей.
type ErrorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// WriteJSON формирует JSON до отправки заголовков.
// При ошибке сериализации клиент получает 500 без частичного тела ответа.
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

// WriteError отправляет код и сообщение ошибки в формате JSON.
func WriteError(w http.ResponseWriter, code int, errCode, message string) {
	WriteJSON(w, code, ErrorResponse{
		Error: ErrorBody{Code: errCode, Message: message},
	})
}

// WriteFieldsError добавляет к ошибке сообщения для отдельных полей.
func WriteFieldsError(w http.ResponseWriter, code int, errCode, message string, fields map[string]string) {
	WriteJSON(w, code, ErrorResponse{
		Error: ErrorBody{Code: errCode, Message: message, Fields: fields},
	})
}
