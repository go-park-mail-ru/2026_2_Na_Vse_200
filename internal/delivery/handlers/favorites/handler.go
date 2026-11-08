package favorites

import (
	"context"
	"errors"
	"log"
	"net/http"
	"uuid"

	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
)

type Repository interface {
	FavoriteIDs(context.Context, models.UserID) ([]string, error)
	SetFavorite(context.Context, models.UserID, models.TrackID, bool) error
}
type Handler struct{ repo Repository }

func NewFavoritesHandler(repo Repository) *Handler { return &Handler{repo: repo} }

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.SessionFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, models.CodeUnauthorized, models.MsgUnauthorized)
		return
	}

	ids, err := h.repo.FavoriteIDs(r.Context(), session.UserID)
	if err != nil {
		log.Printf("list favorites: %v", err)
		response.WriteError(w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
		return
	}

	response.WriteJSON(w, http.StatusOK, struct {
		TrackIDs []string `json:"track_ids"`
	}{ids})
}

func (h *Handler) Set(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.SessionFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, models.CodeUnauthorized, models.MsgUnauthorized)
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.WriteError(w, http.StatusNotFound, models.CodeNotFound, models.MsgNotFound)
		return
	}
	err = h.repo.SetFavorite(r.Context(), session.UserID, models.TrackID(id.String()), r.Method == http.MethodPut)
	if errors.Is(err, models.ErrTrackNotFound) {
		response.WriteError(w, http.StatusNotFound, models.CodeNotFound, models.MsgNotFound)
		return
	}

	if err != nil {
		log.Printf("update favorite: %v", err)
		response.WriteError(w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
