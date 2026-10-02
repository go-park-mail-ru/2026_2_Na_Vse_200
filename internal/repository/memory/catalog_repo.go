package memory

import (
	"context"
	"sync"
	"time"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
)

// CatalogRepo хранит тестовый каталог главной страницы в памяти.
// Данные задаются при создании и дальше только читаются.
type CatalogRepo struct {
	mu      sync.RWMutex
	tracks  []models.Track
	artists []models.Artist
	albums  []models.Album
}

// NewCatalogRepo создаёт хранилище с заранее заполненным каталогом для РК1.
func NewCatalogRepo() *CatalogRepo {
	mira := models.ArtistRef{
		ID:   "c41d7f02-9b63-4a58-8e17-0d5b2a6c9e34",
		Name: "Mira Sol",
	}
	weekenders := models.ArtistRef{
		ID:   "a17e2b44-6c90-4f1d-9e28-3d5a8b7c0142",
		Name: "The Weekenders",
	}
	luna := models.ArtistRef{
		ID:   "f6d0a8c1-2e47-4b9f-91d3-8c5e0a7b3641",
		Name: "Luna Park",
	}

	releaseNorth := time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC)
	releaseCity := time.Date(2023, 3, 10, 0, 0, 0, 0, time.UTC)
	releaseNight := time.Date(2025, 5, 25, 0, 0, 0, 0, time.UTC)

	return &CatalogRepo{
		// Порядок уже «новые первыми»: limit просто отрезает хвост.
		tracks: []models.Track{
			{
				ID:         "3b8e5c90-77a1-4d2f-b6e4-12c9f0a5d738",
				Title:      "Летний дождь",
				DurationMS: 214000,
				CoverURL:   "https://cdn.example.com/covers/1001.jpg",
				Artists:    []models.ArtistRef{mira},
			},
			{
				ID:         "91c2d4e6-0a3b-4f87-9c15-6e8d2a4b7f01",
				Title:      "Северный ветер",
				DurationMS: 284000,
				CoverURL:   "https://cdn.example.com/covers/1002.jpg",
				Artists:    []models.ArtistRef{mira},
			},
			{
				ID:         "7e4a1b2c-5d68-4e90-a123-9f0b8c7d6e54",
				Title:      "На выходных",
				DurationMS: 231000,
				CoverURL:   "https://cdn.example.com/covers/1003.jpg",
				Artists:    []models.ArtistRef{weekenders},
			},
			{
				ID:         "2f9c8d7e-6b5a-4c31-8e20-1d0f9a8b7c65",
				Title:      "Ночной парк",
				DurationMS: 208000,
				CoverURL:   "https://cdn.example.com/covers/1004.jpg",
				Artists:    []models.ArtistRef{luna},
			},
			{
				ID:         "5a6b7c8d-9e0f-4a12-b345-6789abcdef01",
				Title:      "Тихий час",
				DurationMS: 268000,
				CoverURL:   "",
				Artists:    []models.ArtistRef{mira},
			},
			{
				ID:         "0d1e2f3a-4b5c-6d7e-8f90-a1b2c3d4e5f6",
				Title:      "Городской шум",
				DurationMS: 245000,
				CoverURL:   "https://cdn.example.com/covers/1006.jpg",
				Artists:    []models.ArtistRef{weekenders},
			},
		},
		artists: []models.Artist{
			{
				ID:       mira.ID,
				Name:     mira.Name,
				ImageURL: "https://cdn.example.com/artists/c41d7f02.jpg",
			},
			{
				ID:       weekenders.ID,
				Name:     weekenders.Name,
				ImageURL: "https://cdn.example.com/artists/a17e2b44.jpg",
			},
			{
				ID:       luna.ID,
				Name:     luna.Name,
				ImageURL: "https://cdn.example.com/artists/f6d0a8c1.jpg",
			},
		},
		albums: []models.Album{
			{
				ID:          "5e0a94c6-2f18-4b7d-a390-6c81e7f2b405",
				Title:       "Север",
				ReleaseDate: &releaseNorth,
				CoverURL:    "https://cdn.example.com/albums/300.jpg",
				Artists:     []models.ArtistRef{mira},
			},
			{
				ID:          "8b1c2d3e-4f50-4a61-b272-8394a5b6c7d8",
				Title:       "Выходные",
				ReleaseDate: &releaseCity,
				CoverURL:    "https://cdn.example.com/albums/301.jpg",
				Artists:     []models.ArtistRef{weekenders},
			},
			{
				ID:          "9c2d3e4f-5a60-4b71-c382-9405b6c7d8e9",
				Title:       "После полуночи",
				ReleaseDate: &releaseNight,
				CoverURL:    "https://cdn.example.com/albums/302.jpg",
				Artists:     []models.ArtistRef{luna},
			},
		},
	}
}

// HomeTracks возвращает не более limit треков. Пустой каталог — пустой срез.
func (c *CatalogRepo) HomeTracks(ctx context.Context, limit int) ([]models.Track, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return takeTracks(c.tracks, limit), nil
}

// HomeArtists возвращает не более limit исполнителей.
func (c *CatalogRepo) HomeArtists(ctx context.Context, limit int) ([]models.Artist, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return takeArtists(c.artists, limit), nil
}

// HomeAlbums возвращает не более limit альбомов.
func (c *CatalogRepo) HomeAlbums(ctx context.Context, limit int) ([]models.Album, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return takeAlbums(c.albums, limit), nil
}

func takeTracks(items []models.Track, limit int) []models.Track {
	n := clampLimit(limit, len(items))
	out := make([]models.Track, n)
	copy(out, items[:n])
	return out
}

func takeArtists(items []models.Artist, limit int) []models.Artist {
	n := clampLimit(limit, len(items))
	out := make([]models.Artist, n)
	copy(out, items[:n])
	return out
}

func takeAlbums(items []models.Album, limit int) []models.Album {
	n := clampLimit(limit, len(items))
	out := make([]models.Album, n)
	copy(out, items[:n])
	return out
}

func clampLimit(limit, length int) int {
	if limit < 0 {
		return 0
	}
	if limit > length {
		return length
	}
	return limit
}
