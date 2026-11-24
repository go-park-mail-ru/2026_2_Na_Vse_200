package home

import (
	"time"

	"2026_2_Na_Vse_200/internal/models"
)

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
