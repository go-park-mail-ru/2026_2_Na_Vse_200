package artist

import (
	"context"
	"fmt"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
)

const pageLimit = 50

type Page struct {
	Artist models.Artist
	Tracks []models.Track
	Albums []models.Album
}

type Usecase struct {
	artists ports.ArtistRepository
}

func New(artists ports.ArtistRepository) *Usecase {
	return &Usecase{artists: artists}
}

func (u *Usecase) Page(ctx context.Context, id models.ArtistID) (Page, error) {
	artist, err := u.artists.GetByID(ctx, id)
	if err != nil {
		return Page{}, fmt.Errorf("read artist: %w", err)
	}

	tracks, err := u.artists.Tracks(ctx, id, pageLimit, 0)
	if err != nil {
		return Page{}, fmt.Errorf("read artist tracks: %w", err)
	}

	albums, err := u.artists.Albums(ctx, id, pageLimit, 0)
	if err != nil {
		return Page{}, fmt.Errorf("read artist albums: %w", err)
	}

	return Page{Artist: artist, Tracks: tracks, Albums: albums}, nil
}
