package artist

import (
	"fmt"
	"time"
	"uuid"

	"2026_2_Na_Vse_200/internal/models"
	artistusecase "2026_2_Na_Vse_200/internal/usecase/artist"
)

type pageResponse struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Biography string          `json:"biography"`
	ImageURL  *string         `json:"image_url"`
	Tracks    []trackResponse `json:"tracks"`
	Albums    []albumResponse `json:"albums"`
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

func newPageResponse(page artistusecase.ArtistPage) (pageResponse, error) {
	id, err := idToString(string(page.Artist.ID))
	if err != nil {
		return pageResponse{}, fmt.Errorf("artist id: %w", err)
	}
	tracks, err := newTrackResponses(page.Tracks)
	if err != nil {
		return pageResponse{}, err
	}
	albums, err := newAlbumResponses(page.Albums)
	if err != nil {
		return pageResponse{}, err
	}
	return pageResponse{
		ID:        id,
		Name:      page.Artist.Name,
		Biography: page.Artist.Biography,
		ImageURL:  optionalString(page.Artist.ImageURL),
		Tracks:    tracks,
		Albums:    albums,
	}, nil
}

func newTrackResponses(tracks []models.Track) ([]trackResponse, error) {
	out := make([]trackResponse, 0, len(tracks))
	for _, track := range tracks {
		id, err := idToString(string(track.ID))
		if err != nil {
			return nil, fmt.Errorf("track id: %w", err)
		}
		artists, err := newArtistRefResponses(track.Artists)
		if err != nil {
			return nil, err
		}
		out = append(out, trackResponse{
			ID:         id,
			Title:      track.Title,
			DurationMS: track.DurationMS,
			CoverURL:   optionalString(track.CoverURL),
			Artists:    artists,
		})
	}
	return out, nil
}

func newAlbumResponses(albums []models.Album) ([]albumResponse, error) {
	out := make([]albumResponse, 0, len(albums))
	for _, album := range albums {
		id, err := idToString(string(album.ID))
		if err != nil {
			return nil, fmt.Errorf("album id: %w", err)
		}
		artists, err := newArtistRefResponses(album.Artists)
		if err != nil {
			return nil, err
		}
		out = append(out, albumResponse{
			ID:          id,
			Title:       album.Title,
			ReleaseDate: formatReleaseDate(album.ReleaseDate),
			CoverURL:    optionalString(album.CoverURL),
			Artists:     artists,
		})
	}
	return out, nil
}

func newArtistRefResponses(artists []models.ArtistRef) ([]artistRefResponse, error) {
	out := make([]artistRefResponse, 0, len(artists))
	for _, artist := range artists {
		id, err := idToString(string(artist.ID))
		if err != nil {
			return nil, fmt.Errorf("artist ref id: %w", err)
		}
		out = append(out, artistRefResponse{
			ID:   id,
			Name: artist.Name,
		})
	}
	return out, nil
}

func idToString(raw string) (string, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return "", err
	}
	return id.String(), nil
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
