package handlers

import (
	"net/http"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/pkg/response"
)

// healthResponse — тело ответа /health, зафиксировано в docs/api.md, раздел 6.6.
type healthResponse struct {
	Status string `json:"status"`
}

// Health отвечает, что сервис жив. Ручка живёт вне /api/v1 и нужна мониторингу
// и деплою: логики в ней нет намеренно, важен сам факт ответа.
func (a *API) Health(w http.ResponseWriter, r *http.Request) {
	response.WriteJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}
