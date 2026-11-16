package artist

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"2026_2_Na_Vse_200/internal/logger"
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
	return u.PageAt(ctx, id, pageLimit, 0)
}

func (u *ArtistUsecase) PageAt(ctx context.Context, id models.ArtistID, limit, offset int) (ArtistPage, error) {
	logger.UsecaseStarted(ctx, "artists.page")
	if limit < 1 || limit > 100 || offset < 0 {
		return ArtistPage{}, fmt.Errorf("invalid pagination")
	}
	log := zerolog.Ctx(ctx)

	artist, err := u.artists.GetByID(ctx, id)
	if err != nil {
		log.Error().Err(err).Str("artist_id", string(id)).Msg("artist.Page: чтение исполнителя")
		return ArtistPage{}, fmt.Errorf("read artist: %w", err)
	}
	log.Info().Str("artist_id", string(artist.ID)).Msg("artist.Page: исполнитель получен")

	tracks, err := u.artists.Tracks(ctx, id, limit, offset)
	if err != nil {
		log.Error().Err(err).Str("artist_id", string(id)).Msg("artist.Page: чтение треков")
		return ArtistPage{}, fmt.Errorf("read artist tracks: %w", err)
	}
	log.Info().Str("artist_id", string(id)).Int("count", len(tracks)).Msg("artist.Page: треки получены")

	albums, err := u.artists.Albums(ctx, id, limit, offset)
	if err != nil {
		log.Error().Err(err).Str("artist_id", string(id)).Msg("artist.Page: чтение альбомов")
		return ArtistPage{}, fmt.Errorf("read artist albums: %w", err)
	}
	log.Info().Str("artist_id", string(id)).Int("count", len(albums)).Msg("artist.Page: альбомы получены")

	return ArtistPage{Artist: artist, Tracks: tracks, Albums: albums}, nil
}
