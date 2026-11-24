package home

import (
	"time"

	"2026_2_Na_Vse_200/internal/models"
)

// NewTrackResponses converts catalog tracks to their public representation.
func NewTrackResponses(tracks []models.Track) []TrackResponse {
	out := make([]TrackResponse, 0, len(tracks))
	for _, track := range tracks {
		out = append(out, TrackResponse{
			ID:         string(track.ID),
			Title:      track.Title,
			DurationMS: track.DurationMS,
			CoverURL:   optionalString(track.CoverURL),
			Artists:    newArtistRefResponses(track.Artists),
		})
	}
	return out
}

// NewArtistResponses converts catalog artists to their public representation.
func NewArtistResponses(artists []models.Artist) []ArtistResponse {
	out := make([]ArtistResponse, 0, len(artists))
	for _, artist := range artists {
		out = append(out, ArtistResponse{
			ID:       string(artist.ID),
			Name:     artist.Name,
			ImageURL: optionalString(artist.ImageURL),
		})
	}
	return out
}

// NewAlbumResponses converts catalog albums to their public representation.
func NewAlbumResponses(albums []models.Album) []AlbumResponse {
	out := make([]AlbumResponse, 0, len(albums))
	for _, album := range albums {
		out = append(out, NewAlbumResponse(album))
	}
	return out
}

// NewAlbumResponse converts an album to its public metadata.
func NewAlbumResponse(album models.Album) AlbumResponse {
	return AlbumResponse{
		ID:          string(album.ID),
		Title:       album.Title,
		ReleaseDate: formatReleaseDate(album.ReleaseDate),
		CoverURL:    optionalString(album.CoverURL),
		Artists:     newArtistRefResponses(album.Artists),
	}
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
