package memory

import (
	"context"
	"testing"
)

func TestCatalogHomeSeed(t *testing.T) {
	ctx := context.Background()
	repo := NewCatalogRepo()

	tracks, err := repo.HomeTracks(ctx, 20)
	if err != nil {
		t.Fatalf("HomeTracks: %v", err)
	}

	if len(tracks) == 0 {
		t.Fatal("HomeTracks: ожидались seed-треки")
	}

	if len(tracks[0].Artists) == 0 {
		t.Fatal("HomeTracks: у трека должен быть хотя бы один исполнитель")
	}

	artists, err := repo.HomeArtists(ctx, 20)
	if err != nil {
		t.Fatalf("HomeArtists: %v", err)
	}

	if len(artists) == 0 {
		t.Fatal("HomeArtists: ожидались seed-исполнители")
	}

	albums, err := repo.HomeAlbums(ctx, 20)
	if err != nil {
		t.Fatalf("HomeAlbums: %v", err)
	}

	if len(albums) == 0 {
		t.Fatal("HomeAlbums: ожидались seed-альбомы")
	}
}

func TestCatalogHomeLimit(t *testing.T) {
	ctx := context.Background()
	repo := NewCatalogRepo()

	tracks, err := repo.HomeTracks(ctx, 2)
	if err != nil {
		t.Fatalf("HomeTracks: %v", err)
	}

	if len(tracks) != 2 {
		t.Fatalf("HomeTracks: len = %d, ожидалось 2", len(tracks))
	}

	empty, err := repo.HomeArtists(ctx, 0)
	if err != nil {
		t.Fatalf("HomeArtists: %v", err)
	}

	if empty == nil {
		t.Fatal("HomeArtists(0): срез не должен быть nil")
	}

	if len(empty) != 0 {
		t.Fatalf("HomeArtists(0): len = %d, ожидалось 0", len(empty))
	}
}

// Предел запрошенных карточек не должен ломать выдачу: ноль и отрицательное
// дают пустой список, слишком большое — всё, что есть.
func TestCatalogLimits(t *testing.T) {
	ctx := context.Background()
	repo := NewCatalogRepo()

	all, err := repo.HomeTracks(ctx, 1000)
	if err != nil {
		t.Fatalf("HomeTracks: неожиданная ошибка: %v", err)
	}

	if len(all) == 0 {
		t.Fatal("каталог пуст")
	}

	tests := []struct {
		name  string
		limit int
		want  int
	}{
		{name: "отрицательный", limit: -1, want: 0},
		{name: "ноль", limit: 0, want: 0},
		{name: "один", limit: 1, want: 1},
		{name: "больше, чем есть", limit: 1000, want: len(all)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracks, err := repo.HomeTracks(ctx, tt.limit)
			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}

			if len(tracks) != tt.want {
				t.Errorf("треков = %d, ожидалось %d", len(tracks), tt.want)
			}
		})
	}
}
