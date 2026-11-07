package artist

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"uuid"

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
		response.WriteError(w, http.StatusNotFound, models.CodeNotFound, models.MsgNotFound)
		return
	}

	var page artistusecase.ArtistPage
	if r.URL.Query().Has("limit") || r.URL.Query().Has("offset") {
		limit, offset := 100, 0
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
		paged, ok := h.usecase.(interface {
			PageAt(context.Context, models.ArtistID, int, int) (artistusecase.ArtistPage, error)
		})
		if !ok {
			response.WriteError(w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
			return
		}
		page, err = paged.PageAt(r.Context(), models.ArtistID(id.String()), limit, offset)
	} else {
		page, err = h.usecase.Page(r.Context(), models.ArtistID(id.String()))
	}
	switch {
	case errors.Is(err, models.ErrArtistNotFound):
		response.WriteError(w, http.StatusNotFound, models.CodeNotFound, models.MsgNotFound)
		return
	case err != nil:
		log.Printf("чтение страницы исполнителя: %v", err)
		response.WriteError(w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
		return
	}

	body, err := newPageResponse(page)
	if err != nil {
		log.Printf("сборка ответа исполнителя: %v", err)
		response.WriteError(w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
		return
	}
	response.WriteJSON(w, http.StatusOK, body)
}
