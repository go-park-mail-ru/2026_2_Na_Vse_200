package handlers

import "net/http"

// healthResponse — тело ответа /health, зафиксировано в docs/api.md, раздел 6.6.
type healthResponse struct {
	Status string `json:"status"`
}

// Health отвечает, что сервис жив. Ручка живёт вне /api/v1 и нужна мониторингу
// и деплою: логики в ней нет намеренно, важен сам факт ответа.
func (a *API) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}
