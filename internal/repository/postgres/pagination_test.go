package postgres_test

import (
	"context"
	"testing"

	"2026_2_Na_Vse_200/internal/repository/postgres"
	"2026_2_Na_Vse_200/internal/validation"
)

func TestArtistRepositoryRejectsPaginationBeforeQuery(t *testing.T) {
	repo := postgres.NewArtistRepo(nil)
	for _, input := range [][2]int{{0, 0}, {validation.MaxPageLimit + 1, 0}, {1, -1}, {1, validation.MaxPageOffset + 1}} {
		if _, err := repo.Tracks(context.Background(), "artist", input[0], input[1]); err == nil {
			t.Errorf("Tracks accepted %v", input)
		}
		if _, err := repo.Albums(context.Background(), "artist", input[0], input[1]); err == nil {
			t.Errorf("Albums accepted %v", input)
		}
	}
}
