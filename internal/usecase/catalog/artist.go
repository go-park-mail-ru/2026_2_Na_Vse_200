package catalog

import (
	"context"
	"fmt"
	"github.com/rs/zerolog"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
	"2026_2_Na_Vse_200/internal/validation"
)

type ArtistPage struct {
	Artist models.Artist
	Tracks []models.Track
	Albums []models.Album
}

type ArtistUsecase struct{ artists ports.ArtistRepository }

func NewArtistUsecase(artists ports.ArtistRepository) *ArtistUsecase {
	return &ArtistUsecase{artists: artists}
}

func (u *ArtistUsecase) Artist(ctx context.Context, id models.ArtistID, limit, offset int) (ArtistPage, error) {
	log := zerolog.Ctx(ctx)
	log.Info().Str("layer", "usecase").Str("operation", "catalog.artist").Msg("начало сценария")
	if !validation.ValidPagination(limit, offset) {
		return ArtistPage{}, fmt.Errorf("invalid pagination")
	}
	artist, err := u.artists.GetByID(ctx, id)
	if err != nil {
		return ArtistPage{}, fmt.Errorf("artist: %w", err)
	}
	tracks, err := u.artists.Tracks(ctx, id, limit, offset)
	if err != nil {
		return ArtistPage{}, fmt.Errorf("artist tracks: %w", err)
	}
	albums, err := u.artists.Albums(ctx, id, limit, offset)
	if err != nil {
		return ArtistPage{}, fmt.Errorf("artist albums: %w", err)
	}
	return ArtistPage{Artist: artist, Tracks: tracks, Albums: albums}, nil
}
