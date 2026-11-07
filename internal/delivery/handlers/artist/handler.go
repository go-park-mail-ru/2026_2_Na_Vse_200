package artist

import (
	"context"
	"errors"
	"net/http"
	"uuid"

	"github.com/rs/zerolog"

	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
	artistusecase "2026_2_Na_Vse_200/internal/usecase/artist"
)

type ArtistUsecase interface {
	Page(ctx context.Context, id models.ArtistID) (artistusecase.ArtistPage, error)
}

type ArtistHandler struct {
	usecase ArtistUsecase
}

func NewArtistHandler(usecase ArtistUsecase) *ArtistHandler {
	return &ArtistHandler{usecase: usecase}
}

func (h *ArtistHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.WriteError(r.Context(), w, http.StatusNotFound, models.CodeNotFound, models.MsgNotFound)
		return
	}

	page, err := h.usecase.Page(r.Context(), models.ArtistID(id.String()))
	switch {
	case errors.Is(err, models.ErrArtistNotFound):
		response.WriteError(r.Context(), w, http.StatusNotFound, models.CodeNotFound, models.MsgNotFound)
		return
	case err != nil:
		zerolog.Ctx(r.Context()).Error().Err(err).Str("operation", "artist.page").Msg("чтение страницы исполнителя")
		response.WriteError(r.Context(), w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
		return
	}

	body, err := newPageResponse(page)
	if err != nil {
		zerolog.Ctx(r.Context()).Error().Err(err).Str("operation", "artist.response").Msg("сборка ответа исполнителя")
		response.WriteError(r.Context(), w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
		return
	}
	response.WriteJSON(r.Context(), w, http.StatusOK, body)
}
