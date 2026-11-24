package home

import (
	"context"
	"net/http"

	"github.com/rs/zerolog"

	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
	catalogusecase "2026_2_Na_Vse_200/internal/usecase/catalog"
)

type CatalogUsecase interface {
	Home(context.Context) (catalogusecase.Home, error)
}

type Handler struct {
	catalog CatalogUsecase
}

func New(catalog CatalogUsecase) *Handler {
	return &Handler{catalog: catalog}
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	result, err := h.catalog.Home(r.Context())
	if err != nil {
		zerolog.Ctx(r.Context()).Error().Err(err).Str("operation", "catalog.home").Msg("ошибка чтения каталога")
		response.WriteError(r.Context(), w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
		return
	}
	response.WriteJSON(r.Context(), w, http.StatusOK, homeResponse{
		Tracks:  newTrackResponses(result.Tracks),
		Artists: newArtistResponses(result.Artists),
		Albums:  newAlbumResponses(result.Albums),
	})
}
