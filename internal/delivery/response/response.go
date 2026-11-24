package response

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog"
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
func WriteJSON(ctx context.Context, w http.ResponseWriter, code int, data any) {
	body, err := json.Marshal(data)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Str("operation", "response.marshal").Msg("сборка JSON-ответа")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)

	if _, err := w.Write(body); err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Str("operation", "response.write").Msg("отправка ответа")
	}
}

// WriteError отправляет код и сообщение ошибки в формате JSON.
func WriteError(ctx context.Context, w http.ResponseWriter, code int, errCode, message string) {
	WriteJSON(ctx, w, code, ErrorResponse{
		Error: ErrorBody{Code: errCode, Message: message},
	})
}

// WriteFieldsError добавляет к ошибке сообщения для отдельных полей.
func WriteFieldsError(ctx context.Context, w http.ResponseWriter, code int, errCode, message string, fields map[string]string) {
	WriteJSON(ctx, w, code, ErrorResponse{
		Error: ErrorBody{Code: errCode, Message: message, Fields: fields},
	})
}
