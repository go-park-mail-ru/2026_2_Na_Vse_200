package artist

import (
	"context"
	"errors"
	"testing"

	"2026_2_Na_Vse_200/internal/models"
)

type artistRepoStub struct {
	artist models.Artist
	tracks []models.Track
	albums []models.Album
	fail   string
	err    error
	calls  []string
}

func (s *artistRepoStub) GetByID(context.Context, models.ArtistID) (models.Artist, error) {
	s.calls = append(s.calls, "get")
	if s.fail == "get" {
		return models.Artist{}, s.err
	}
	return s.artist, nil
}

func (s *artistRepoStub) Tracks(context.Context, models.ArtistID, int, int) ([]models.Track, error) {
	s.calls = append(s.calls, "tracks")
	if s.fail == "tracks" {
		return nil, s.err
	}
	return s.tracks, nil
}

func (s *artistRepoStub) Albums(context.Context, models.ArtistID, int, int) ([]models.Album, error) {
	s.calls = append(s.calls, "albums")
	if s.fail == "albums" {
		return nil, s.err
	}
	return s.albums, nil
}

func TestPage(t *testing.T) {
	id := models.ArtistID("c41d7f02-9b63-4a58-8e17-0d5b2a6c9e34")
	repo := &artistRepoStub{
		artist: models.Artist{ID: id, Name: "Кино", Biography: "группа"},
		tracks: []models.Track{{ID: "track-1", Title: "Группа крови"}},
		albums: []models.Album{{ID: "album-1", Title: "Группа крови"}},
	}

	page, err := New(repo).Page(context.Background(), id)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if page.Artist.Name != "Кино" || len(page.Tracks) != 1 || len(page.Albums) != 1 {
		t.Fatalf("page = %+v", page)
	}
}

func TestPageErrors(t *testing.T) {
	failure := errors.New("db down")
	for _, fail := range []string{"get", "tracks", "albums"} {
		t.Run(fail, func(t *testing.T) {
			repo := &artistRepoStub{fail: fail, err: failure, artist: models.Artist{ID: "a"}}
			_, err := New(repo).Page(context.Background(), "a")
			if !errors.Is(err, failure) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}
