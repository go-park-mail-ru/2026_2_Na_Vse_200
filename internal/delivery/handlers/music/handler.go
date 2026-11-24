package music

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"uuid"

	"2026_2_Na_Vse_200/internal/delivery/handlers/home"
	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
	"github.com/rs/zerolog"
)

// Repository reads paginated search results and album details.
type Repository interface {
	Search(context.Context, models.SearchQuery) (models.SearchResult, error)
	Album(context.Context, models.AlbumID) (models.AlbumPage, error)
}

// Handler serves catalog search and album pages.
type Handler struct{ repo Repository }

// New creates a catalog handler.
func New(repo Repository) *Handler { return &Handler{repo: repo} }

// Search responds with results for the requested catalog entity kinds.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	kind := models.SearchKind(values.Get("type"))
	if kind == "" {
		kind = models.SearchAll
	}
	if kind != models.SearchAll && kind != models.SearchTracks && kind != models.SearchArtists && kind != models.SearchAlbums {
		response.WriteError(r.Context(), w, http.StatusBadRequest, models.CodeValidationFailed, "Некорректный тип поиска.")
		return
	}
	limit, limitErr := paginationValue(values.Get("limit"), 20)
	offset, offsetErr := paginationValue(values.Get("offset"), 0)
	if limitErr != nil || offsetErr != nil || limit < 1 || limit > 100 || offset < 0 {
		response.WriteError(r.Context(), w, http.StatusBadRequest, "invalid_pagination", "limit: от 1 до 100, offset: не меньше 0")
		return
	}
	query := strings.TrimSpace(values.Get("q"))
	var result models.SearchResult
	if query != "" || kind == models.SearchTracks {
		var err error
		result, err = h.repo.Search(r.Context(), models.SearchQuery{
			Text:   query,
			Kind:   kind,
			Limit:  limit + 1,
			Offset: offset,
		})
		if err != nil {
			zerolog.Ctx(r.Context()).Error().Err(err).Str("operation", "catalog.search").Msg("search catalog")
			response.WriteError(r.Context(), w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
			return
		}
	}
	response.WriteJSON(r.Context(), w, http.StatusOK, struct {
		Tracks  []home.TrackResponse  `json:"tracks"`
		Artists []home.ArtistResponse `json:"artists"`
		Albums  []home.AlbumResponse  `json:"albums"`
		HasMore map[string]bool       `json:"has_more"`
	}{
		Tracks:  home.NewTrackResponses(result.Tracks[:min(limit, len(result.Tracks))]),
		Artists: home.NewArtistResponses(result.Artists[:min(limit, len(result.Artists))]),
		Albums:  home.NewAlbumResponses(result.Albums[:min(limit, len(result.Albums))]),
		HasMore: map[string]bool{
			"tracks":  len(result.Tracks) > limit,
			"artists": len(result.Artists) > limit,
			"albums":  len(result.Albums) > limit,
		},
	})
}

func paginationValue(raw string, fallback int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	return strconv.Atoi(raw)
}

// Album responds with album metadata and its ordered published tracks.
func (h *Handler) Album(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.WriteError(r.Context(), w, http.StatusNotFound, models.CodeNotFound, models.MsgNotFound)
		return
	}
	page, err := h.repo.Album(r.Context(), models.AlbumID(id.String()))
	switch {
	case errors.Is(err, models.ErrAlbumNotFound):
		response.WriteError(r.Context(), w, http.StatusNotFound, models.CodeNotFound, models.MsgNotFound)
	case err != nil:
		zerolog.Ctx(r.Context()).Error().Err(err).Str("operation", "catalog.album").Msg("read album page")
		response.WriteError(r.Context(), w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
	default:
		response.WriteJSON(r.Context(), w, http.StatusOK, struct {
			home.AlbumResponse

			Tracks []home.TrackResponse `json:"tracks"`
		}{AlbumResponse: home.NewAlbumResponse(page.Album), Tracks: home.NewTrackResponses(page.Tracks)})
	}
}
