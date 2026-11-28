package catalog

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
)

const homeLimit = 20

type Home struct {
	Tracks  []models.Track
	Artists []models.Artist
	Albums  []models.Album
}

// PopularTracks ограничивает витрину отдельно от очереди воспроизведения.
func (h Home) PopularTracks(limit int, enabled bool) []models.Track {
	if !enabled || limit <= 0 {
		return nil
	}
	return h.Tracks[:min(limit, len(h.Tracks))]
}

type Usecase struct {
	catalog ports.CatalogRepository
}

func New(catalog ports.CatalogRepository) *Usecase {
	return &Usecase{catalog: catalog}
}

func (u *Usecase) Home(ctx context.Context) (Home, error) {
	log := zerolog.Ctx(ctx)
	log.Info().Str("layer", "usecase").Str("operation", "catalog.home").Msg("начало сценария")

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
