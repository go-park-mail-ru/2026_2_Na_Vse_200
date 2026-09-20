package handlers

import "net/http"

// MainPage отправляет приветственное JSON-сообщение для пути /
// Для остальных путей возвращает статус 404
func MainPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, "это главная страница")
}
