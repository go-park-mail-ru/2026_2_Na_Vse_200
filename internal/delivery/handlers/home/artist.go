package home

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"

	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
	catalogusecase "2026_2_Na_Vse_200/internal/usecase/catalog"
)

type ArtistUsecase interface {
	Artist(context.Context, models.ArtistID, int, int) (catalogusecase.ArtistPage, error)
}

type ArtistHandler struct{ artists ArtistUsecase }

func NewArtistHandler(artists ArtistUsecase) *ArtistHandler { return &ArtistHandler{artists: artists} }

func (h *ArtistHandler) Artist(w http.ResponseWriter, r *http.Request) {
	limit, offset := 100, 0
	var err error
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
	}
	if err == nil {
		if raw := r.URL.Query().Get("offset"); raw != "" {
			offset, err = strconv.Atoi(raw)
		}
	}
	if err != nil || limit < 1 || limit > 100 || offset < 0 {
		response.WriteError(w, http.StatusBadRequest, "invalid_pagination", "limit: от 1 до 100; offset: не меньше 0")
		return
	}
	result, err := h.artists.Artist(r.Context(), models.ArtistID(r.PathValue("id")), limit, offset)
	if errors.Is(err, models.ErrArtistNotFound) {
		response.WriteError(w, http.StatusNotFound, models.CodeNotFound, "Исполнитель не найден")
		return
	}
	if err != nil {
		log.Printf("ошибка чтения исполнителя: %v", err)
		response.WriteError(w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
		return
	}
	response.WriteJSON(w, http.StatusOK, struct {
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
