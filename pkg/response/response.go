// Package response пишет HTTP-ответы в формате, зафиксированном контрактом API
// (docs/api.md, раздел 1).
//
// Пакет лежит в pkg, а не в internal: он не знает ни про пользователей, ни про
// каталог — только про то, как выглядит успешный ответ и ошибка. Им одинаково
// пользуются обработчики и обвязка (middleware), поэтому он не может жить
// внутри ни одного из них: иначе зависимость пошла бы в обратную сторону.
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

// ErrorBody — тело ошибки. Fields заполняется только при ошибках валидации,
// поэтому помечено omitempty: в остальных ответах ключа просто не будет.
type ErrorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// WriteJSON отправляет data как JSON с указанным статусом.
//
// Порядок важен: сначала заголовки, потом WriteHeader, потом тело. После первой
// записи статус поменять уже нельзя, а без WriteHeader ушёл бы 200.
// Поэтому JSON сначала собирается целиком: если Marshal упадёт, мы ещё успеем
// отдать 500, а не половину ответа после отправленного статуса.
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
		// Клиент разорвал соединение: сказать ему уже нечего, пишем себе в лог.
		log.Printf("отправка ответа: %v", err)
	}
}

// WriteError отправляет ошибку в едином формате API.
// message — текст на русском, его можно показать пользователю.
func WriteError(w http.ResponseWriter, code int, errCode, message string) {
	WriteJSON(w, code, ErrorResponse{
		Error: ErrorBody{
			Code:    errCode,
			Message: message,
		},
	})
}

// WriteValidationError отправляет 400 с разбором по полям: ключ — имя поля
// из запроса, значение — что с ним не так. Фронтенд подсвечивает все поля сразу.
func WriteValidationError(w http.ResponseWriter, fields map[string]string) {
	WriteJSON(w, http.StatusBadRequest, ErrorResponse{
		Error: ErrorBody{
			Code:    apimessage.CodeValidationFailed,
			Message: apimessage.MsgValidationFailed,
			Fields:  fields,
		},
	})
}

// WriteInternalError логирует настоящую причину у себя и отдаёт клиенту общий
// текст. Детали (SQL, пути, стектрейсы) наружу не уходят никогда.
func WriteInternalError(w http.ResponseWriter, context string, err error) {
	log.Printf("%s: %v", context, err)
	WriteError(w, http.StatusInternalServerError, apimessage.CodeInternal, apimessage.MsgInternal)
}
