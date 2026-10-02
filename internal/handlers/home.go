package handlers

import (
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/pkg/response"
)

const _homeLimit = 20

type homeResponse struct {
	Tracks  []trackResponse  `json:"tracks"`
	Artists []artistResponse `json:"artists"`
	Albums  []albumResponse  `json:"albums"`
}

type trackResponse struct {
	ID         string              `json:"id"`
	Title      string              `json:"title"`
	DurationMS int                 `json:"duration_ms"`
	CoverURL   *string             `json:"cover_url"`
	Artists    []artistRefResponse `json:"artists"`
}

type artistResponse struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	ImageURL *string `json:"image_url"`
}

type albumResponse struct {
	ID          string              `json:"id"`
	Title       string              `json:"title"`
	ReleaseDate *string             `json:"release_date"`
	CoverURL    *string             `json:"cover_url"`
	Artists     []artistRefResponse `json:"artists"`
}

type artistRefResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Home отдаёт данные главной страницы: GET /api/v1/home
// Ручка публичная, т.е ответ одинаковый для гостя и для вошедшего пользователя
func (a *API) Home(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tracks, err := a.deps.Catalog.HomeTracks(ctx, _homeLimit)
	if err != nil {
		writeInternalError(w, errReadCatalog, err)
		return
	}

	artists, err := a.deps.Catalog.HomeArtists(ctx, _homeLimit)
	if err != nil {
		writeInternalError(w, errReadCatalog, err)
		return
	}

	albums, err := a.deps.Catalog.HomeAlbums(ctx, _homeLimit)
	if err != nil {
		writeInternalError(w, errReadCatalog, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, homeResponse{
		Tracks:  newTrackResponses(tracks),
		Artists: newArtistResponses(artists),
		Albums:  newAlbumResponses(albums),
	})
}

func newTrackResponses(tracks []models.Track) []trackResponse {
	out := make([]trackResponse, 0, len(tracks))
	for _, track := range tracks {
		out = append(out, trackResponse{
			ID:         string(track.ID),
			Title:      track.Title,
			DurationMS: track.DurationMS,
			CoverURL:   optionalString(track.CoverURL),
			Artists:    newArtistRefResponses(track.Artists),
		})
	}
	return out
}

func newArtistResponses(artists []models.Artist) []artistResponse {
	out := make([]artistResponse, 0, len(artists))
	for _, artist := range artists {
		out = append(out, artistResponse{
			ID:       string(artist.ID),
			Name:     artist.Name,
			ImageURL: optionalString(artist.ImageURL),
		})
	}
	return out
}

func newAlbumResponses(albums []models.Album) []albumResponse {
	out := make([]albumResponse, 0, len(albums))
	for _, album := range albums {
		out = append(out, albumResponse{
			ID:          string(album.ID),
			Title:       album.Title,
			ReleaseDate: formatReleaseDate(album.ReleaseDate),
			CoverURL:    optionalString(album.CoverURL),
			Artists:     newArtistRefResponses(album.Artists),
		})
	}
	return out
}

func newArtistRefResponses(artists []models.ArtistRef) []artistRefResponse {
	out := make([]artistRefResponse, 0, len(artists))
	for _, artist := range artists {
		out = append(out, artistRefResponse{
			ID:   string(artist.ID),
			Name: artist.Name,
		})
	}
	return out
}

// optionalString отдаёт null, если строка пустая
func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// formatReleaseDate отдаёт YYYY-MM-DD или null
func formatReleaseDate(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format("2006-01-02")
	return &formatted
}
