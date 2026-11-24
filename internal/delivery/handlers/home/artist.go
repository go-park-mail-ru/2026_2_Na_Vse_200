package home

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/rs/zerolog"

	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
	catalogusecase "2026_2_Na_Vse_200/internal/usecase/catalog"
	"2026_2_Na_Vse_200/internal/validation"
)

type ArtistUsecase interface {
	Artist(context.Context, models.ArtistID, int, int) (catalogusecase.ArtistPage, error)
}

type ArtistHandler struct{ artists ArtistUsecase }

func NewArtistHandler(artists ArtistUsecase) *ArtistHandler { return &ArtistHandler{artists: artists} }

func (h *ArtistHandler) Artist(w http.ResponseWriter, r *http.Request) {
	limit, offset := validation.DefaultPageLimit, 0
	var err error
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
	}
	if err == nil {
		if raw := r.URL.Query().Get("offset"); raw != "" {
			offset, err = strconv.Atoi(raw)
		}
	}
	if err != nil || !validation.ValidPagination(limit, offset) {
		response.WriteError(r.Context(), w, http.StatusBadRequest, "invalid_pagination", "Некорректные параметры пагинации")
		return
	}
	result, err := h.artists.Artist(r.Context(), models.ArtistID(r.PathValue("id")), limit, offset)
	if errors.Is(err, models.ErrArtistNotFound) {
		response.WriteError(r.Context(), w, http.StatusNotFound, models.CodeNotFound, "Исполнитель не найден")
		return
	}
	if err != nil {
		zerolog.Ctx(r.Context()).Error().Err(err).Str("operation", "artist.page").Msg("ошибка чтения исполнителя")
		response.WriteError(r.Context(), w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
		return
	}
	response.WriteJSON(r.Context(), w, http.StatusOK, struct {
		artistResponse
		Biography string          `json:"biography"`
		Tracks    []trackResponse `json:"tracks"`
		Albums    []albumResponse `json:"albums"`
	}{
		artistResponse: artistResponse{ID: string(result.Artist.ID), Name: result.Artist.Name, ImageURL: optionalString(result.Artist.ImageURL)},
		Biography:      result.Artist.Biography,
		Tracks:         newTrackResponses(result.Tracks),
		Albums:         newAlbumResponses(result.Albums),
	})
}
