package favorites

import (
	"context"
	"errors"
	"net/http"
	"uuid"

	"2026_2_Na_Vse_200/internal/delivery/handlers/home"
	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
	"github.com/rs/zerolog"
)

type Repository interface {
	FavoriteTracks(context.Context, models.UserID) ([]models.Track, error)
	SetFavorite(context.Context, models.UserID, models.TrackID, bool) error
}
type Handler struct{ repo Repository }

func NewFavoritesHandler(repo Repository) *Handler { return &Handler{repo: repo} }

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.SessionFromContext(r.Context())
	if !ok {
		response.WriteError(r.Context(), w, http.StatusUnauthorized, models.CodeUnauthorized, models.MsgUnauthorized)
		return
	}

	tracks, err := h.repo.FavoriteTracks(r.Context(), session.UserID)
	if err != nil {
		zerolog.Ctx(r.Context()).Error().Err(err).Str("operation", "favorites.list").Msg("list favorites")
		response.WriteError(r.Context(), w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
		return
	}

	ids := make([]string, 0, len(tracks))
	for _, track := range tracks {
		ids = append(ids, string(track.ID))
	}
	w.Header().Set("Cache-Control", "private, no-store")
	response.WriteJSON(r.Context(), w, http.StatusOK, struct {
		TrackIDs []string             `json:"track_ids"`
		Tracks   []home.TrackResponse `json:"tracks"`
	}{TrackIDs: ids, Tracks: home.NewTrackResponses(tracks)})
}

func (h *Handler) Set(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.SessionFromContext(r.Context())
	if !ok {
		response.WriteError(r.Context(), w, http.StatusUnauthorized, models.CodeUnauthorized, models.MsgUnauthorized)
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.WriteError(r.Context(), w, http.StatusNotFound, models.CodeNotFound, models.MsgNotFound)
		return
	}
	err = h.repo.SetFavorite(r.Context(), session.UserID, models.TrackID(id.String()), r.Method == http.MethodPut)
	if errors.Is(err, models.ErrTrackNotFound) {
		response.WriteError(r.Context(), w, http.StatusNotFound, models.CodeNotFound, models.MsgNotFound)
		return
	}

	if err != nil {
		zerolog.Ctx(r.Context()).Error().Err(err).Str("operation", "favorites.set").Msg("update favorite")
		response.WriteError(r.Context(), w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
