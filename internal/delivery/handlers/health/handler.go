package health

import (
	"net/http"

	"2026_2_Na_Vse_200/internal/delivery/response"
)

type healthResponse struct {
	Status string `json:"status"`
}

// Health отвечает на проверку доступности сервиса.
func Health(w http.ResponseWriter, r *http.Request) {
	response.WriteJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}
