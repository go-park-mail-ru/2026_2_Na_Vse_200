package catalog

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"2026_2_Na_Vse_200/internal/logger"
	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
)

const homeLimit = 20

type Home struct {
	Tracks  []models.Track
	Artists []models.Artist
	Albums  []models.Album
}

type Usecase struct {
	catalog ports.CatalogRepository
}

func New(catalog ports.CatalogRepository) *Usecase {
	return &Usecase{catalog: catalog}
}

func (u *Usecase) Home(ctx context.Context) (Home, error) {
	logger.UsecaseStarted(ctx, "catalog.home")
	log := zerolog.Ctx(ctx)

	tracks, err := u.catalog.HomeTracks(ctx, homeLimit)
	if err != nil {
		log.Error().Err(err).Msg("catalog.Home: чтение треков")
		return Home{}, fmt.Errorf("home tracks: %w", err)
	}
	log.Info().Int("count", len(tracks)).Msg("catalog.Home: треки получены")

	artists, err := u.catalog.HomeArtists(ctx, homeLimit)
	if err != nil {
		log.Error().Err(err).Msg("catalog.Home: чтение исполнителей")
		return Home{}, fmt.Errorf("home artists: %w", err)
	}
	log.Info().Int("count", len(artists)).Msg("catalog.Home: исполнители получены")

	albums, err := u.catalog.HomeAlbums(ctx, homeLimit)
	if err != nil {
		log.Error().Err(err).Msg("catalog.Home: чтение альбомов")
		return Home{}, fmt.Errorf("home albums: %w", err)
	}
	log.Info().Int("count", len(albums)).Msg("catalog.Home: альбомы получены")

	return Home{Tracks: tracks, Artists: artists, Albums: albums}, nil
}
