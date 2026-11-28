package home

import (
	"context"
	"errors"
	"net/http"

	"github.com/rs/zerolog"

	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/onlineconf"
	catalogusecase "2026_2_Na_Vse_200/internal/usecase/catalog"
)

type CatalogUsecase interface {
	Home(context.Context) (catalogusecase.Home, error)
}

type Handler struct {
	catalog       CatalogUsecase
	cache         *onlineconf.Cache
	sessionCookie string
	sessionByID   middleware.SessionByIDFunc
}

func New(catalog CatalogUsecase, cache *onlineconf.Cache, sessionCookie string, sessionByID middleware.SessionByIDFunc) *Handler {
	return &Handler{catalog: catalog, cache: cache, sessionCookie: sessionCookie, sessionByID: sessionByID}
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	var snapshot *onlineconf.Snapshot
	if h.cache != nil {
		snapshot = h.cache.Snapshot()
	}
	var userID string
	if snapshot.NeedsUser() {
		userID = h.userID(r)
	}
	settings := snapshot.PopularTracks(userID)

	result, err := h.catalog.Home(r.Context())
	if err != nil {
		zerolog.Ctx(r.Context()).Error().Err(err).Str("operation", "catalog.home").Msg("ошибка чтения каталога")
		response.WriteError(r.Context(), w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
		return
	}
	response.WriteJSON(r.Context(), w, http.StatusOK, homeResponse{
		Tracks:        NewTrackResponses(result.Tracks),
		Artists:       NewArtistResponses(result.Artists),
		Albums:        NewAlbumResponses(result.Albums),
		PopularTracks: NewTrackResponses(result.PopularTracks(settings.Limit, settings.Enabled)),
		OnlineConf:    settings,
	})
}

func (h *Handler) userID(r *http.Request) string {
	cookie, err := r.Cookie(h.sessionCookie)
	if err != nil || cookie.Value == "" || h.sessionByID == nil {
		return ""
	}
	session, err := h.sessionByID(r.Context(), models.SessionID(cookie.Value))
	switch {
	case err == nil:
		return string(session.UserID)
	case errors.Is(err, models.ErrSessionNotFound), errors.Is(err, models.ErrSessionExpired):
		return ""
	default:
		zerolog.Ctx(r.Context()).Warn().Err(err).Str("operation", "onlineconf.home_session").Msg("не удалось проверить сессию")
		return ""
	}
}
