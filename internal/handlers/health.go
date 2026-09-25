package handlers

import (
	"net/http"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/pkg/response"
)

type healthResponse struct {
	Status string `json:"status"`
}

// Health сообщает, что сервис жив: GET /health. Нужен мониторингу и деплою.
func (a *API) Health(w http.ResponseWriter, r *http.Request) {
	response.WriteJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}
