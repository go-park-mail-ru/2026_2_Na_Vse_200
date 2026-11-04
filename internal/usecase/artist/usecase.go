package artist

import (
	"context"
	"fmt"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
)

const pageLimit = 50

type ArtistPage struct {
	Artist models.Artist
	Tracks []models.Track
	Albums []models.Album
}

type ArtistUsecase struct {
	artists ports.ArtistRepository
}

func NewArtistUsecase(artists ports.ArtistRepository) *ArtistUsecase {
	return &ArtistUsecase{artists: artists}
}

func (u *ArtistUsecase) Page(ctx context.Context, id models.ArtistID) (ArtistPage, error) {
	artist, err := u.artists.GetByID(ctx, id)
	if err != nil {
		return ArtistPage{}, fmt.Errorf("read artist: %w", err)
	}

	tracks, err := u.artists.Tracks(ctx, id, pageLimit, 0)
	if err != nil {
		return ArtistPage{}, fmt.Errorf("read artist tracks: %w", err)
	}

	albums, err := u.artists.Albums(ctx, id, pageLimit, 0)
	if err != nil {
		return ArtistPage{}, fmt.Errorf("read artist albums: %w", err)
	}

	return ArtistPage{Artist: artist, Tracks: tracks, Albums: albums}, nil
}
