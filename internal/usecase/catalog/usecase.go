// Package catalog implements catalog queries independently of HTTP.
package catalog

import (
	"context"
	"fmt"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
)

const homeLimit = 20

type Home struct {
	Tracks  []models.Track
	Artists []models.Artist
	Albums  []models.Album
}

type Usecase struct{ catalog ports.CatalogRepository }

func New(catalog ports.CatalogRepository) *Usecase { return &Usecase{catalog: catalog} }

func (u *Usecase) Home(ctx context.Context) (Home, error) {
	tracks, err := u.catalog.HomeTracks(ctx, homeLimit)
	if err != nil {
		return Home{}, fmt.Errorf("home tracks: %w", err)
	}
	artists, err := u.catalog.HomeArtists(ctx, homeLimit)
	if err != nil {
		return Home{}, fmt.Errorf("home artists: %w", err)
	}
	albums, err := u.catalog.HomeAlbums(ctx, homeLimit)
	if err != nil {
		return Home{}, fmt.Errorf("home albums: %w", err)
	}
	return Home{Tracks: tracks, Artists: artists, Albums: albums}, nil
}
