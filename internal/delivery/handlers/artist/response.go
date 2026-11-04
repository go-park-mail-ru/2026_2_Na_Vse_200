package artist

import (
	"time"

	"2026_2_Na_Vse_200/internal/models"
	artistusecase "2026_2_Na_Vse_200/internal/usecase/artist"
)

type pageResponse struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Biography  string          `json:"biography"`
	ImageURL   *string         `json:"image_url"`
	Tracks     []trackResponse `json:"tracks"`
	Albums     []albumResponse `json:"albums"`
}

type trackResponse struct {
	ID         string              `json:"id"`
	Title      string              `json:"title"`
	DurationMS int                 `json:"duration_ms"`
	CoverURL   *string             `json:"cover_url"`
	Artists    []artistRefResponse `json:"artists"`
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

func newPageResponse(page artistusecase.Page) pageResponse {
	return pageResponse{
		ID:        string(page.Artist.ID),
		Name:      page.Artist.Name,
		Biography: page.Artist.Biography,
		ImageURL:  optionalString(page.Artist.ImageURL),
		Tracks:    newTrackResponses(page.Tracks),
		Albums:    newAlbumResponses(page.Albums),
	}
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

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func formatReleaseDate(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format("2006-01-02")
	return &formatted
}
