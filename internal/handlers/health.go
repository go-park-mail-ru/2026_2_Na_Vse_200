package handlers

import "net/http"

// Health отправляет JSON-сообщение ок для проверки доступности сервера
func Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, "ок")
}
