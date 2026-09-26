package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// writeJSON записывает message в HTTP ответ как JSON-объект
// с полем message и устанавливает Content-Type.
func writeJSON(w http.ResponseWriter, message string) {
	response := map[string]string{
		"message": message,
	}

	data, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "не удалось сформировать JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, err = w.Write(data)
	if err != nil {
		fmt.Println("ошибка отправки ответа:", err)
	}
}
