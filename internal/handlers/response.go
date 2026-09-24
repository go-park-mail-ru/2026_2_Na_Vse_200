package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

// Машинные коды ошибок из справочника контракта (docs/api.md, раздел 1).
// Фронт ветвит логику по ним, поэтому строки менять нельзя, не обновив контракт.
const (
	CodeInvalidJSON        = "invalid_json"
	CodeValidationFailed   = "validation_failed"
	CodeEmailTaken         = "email_taken"
	CodeInvalidCredentials = "invalid_credentials"
	CodeUnauthorized       = "unauthorized"
	CodeNotFound           = "not_found"
	CodeMethodNotAllowed   = "method_not_allowed"
	CodeInternal           = "internal_error"
)

// errorResponse — внешняя обёртка ошибки: {"error": {...}}.
type errorResponse struct {
	Error errorBody `json:"error"`
}

// errorBody — тело ошибки. Fields заполняется только при ошибках валидации,
// поэтому помечено omitempty: в остальных ответах ключа просто не будет.
type errorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// writeJSON отправляет data как JSON с указанным статусом.
//
// Порядок важен: сначала заголовки, потом WriteHeader, потом тело. После первой
// записи статус поменять уже нельзя, а без WriteHeader ушёл бы 200.
// Поэтому JSON сначала собирается целиком: если Marshal упадёт, мы ещё успеем
// отдать 500, а не половину ответа после отправленного статуса.
func writeJSON(w http.ResponseWriter, code int, data any) {
	body, err := json.Marshal(data)
	if err != nil {
		log.Printf("сборка JSON-ответа: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)

	if _, err := w.Write(body); err != nil {
		// Клиент разорвал соединение: сказать ему уже нечего, пишем себе в лог.
		log.Printf("отправка ответа: %v", err)
	}
}

// writeError отправляет ошибку в едином формате API.
// message — текст на русском, его можно показать пользователю.
func writeError(w http.ResponseWriter, code int, errCode, message string) {
	writeJSON(w, code, errorResponse{
		Error: errorBody{
			Code:    errCode,
			Message: message,
		},
	})
}

// writeValidationError отправляет 400 с разбором по полям: ключ — имя поля
// из запроса, значение — что с ним не так. Фронт подсвечивает все поля сразу.
func writeValidationError(w http.ResponseWriter, fields map[string]string) {
	writeJSON(w, http.StatusBadRequest, errorResponse{
		Error: errorBody{
			Code:    CodeValidationFailed,
			Message: "Проверьте правильность заполнения полей",
			Fields:  fields,
		},
	})
}

// writeInternalError логирует настоящую причину у себя и отдаёт клиенту общий текст.
// Детали (SQL, пути, стектрейсы) наружу не уходят никогда.
func writeInternalError(w http.ResponseWriter, context string, err error) {
	log.Printf("%s: %v", context, err)
	writeError(w, http.StatusInternalServerError, CodeInternal, "Внутренняя ошибка сервера")
}
