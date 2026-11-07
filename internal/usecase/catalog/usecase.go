package catalog

import (
	"context"
	"fmt"
	"log/slog"

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
	log     *slog.Logger
}

func New(catalog ports.CatalogRepository, log *slog.Logger) *Usecase {
	if log == nil {
		log = logger.Discard()
	}
	return &Usecase{catalog: catalog, log: log}
}

func (u *Usecase) Home(ctx context.Context) (Home, error) {
	log := logger.FromContext(ctx, u.log)
	log.Info("catalog.Home: старт")

	tracks, err := u.catalog.HomeTracks(ctx, homeLimit)
	if err != nil {
		log.Error("catalog.Home: чтение треков", slog.Any("error", err))
		return Home{}, fmt.Errorf("home tracks: %w", err)
	}
	log.Info("catalog.Home: треки получены", slog.Int("count", len(tracks)))

	artists, err := u.catalog.HomeArtists(ctx, homeLimit)
	if err != nil {
		log.Error("catalog.Home: чтение исполнителей", slog.Any("error", err))
		return Home{}, fmt.Errorf("home artists: %w", err)
	}
	log.Info("catalog.Home: исполнители получены", slog.Int("count", len(artists)))

	albums, err := u.catalog.HomeAlbums(ctx, homeLimit)
	if err != nil {
		log.Error("catalog.Home: чтение альбомов", slog.Any("error", err))
		return Home{}, fmt.Errorf("home albums: %w", err)
	}
	log.Info("catalog.Home: альбомы получены", slog.Int("count", len(albums)))

	return Home{Tracks: tracks, Artists: artists, Albums: albums}, nil
}
