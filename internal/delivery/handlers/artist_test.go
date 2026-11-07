package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"2026_2_Na_Vse_200/internal/models"
)

type artistPageResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Biography string `json:"biography"`
	ImageURL  *string
	Tracks    []trackResponse `json:"tracks"`
	Albums    []albumResponse `json:"albums"`
}

func newArtistHandler(artists artistRepoStub) http.Handler {
	api := newRepositoryTestAPI(&testConfig, &repositoryTestDeps{
		Users:    &authUsersFixture{},
		Sessions: &authSessionsFixture{},
		Catalog:  homeCatalogFixture{},
		Artists:  artists,
		Hasher:   nil,
	})
	return middleware.Chain(api.Routes(), middleware.WithJSONErrors)
}

func TestArtistPageSuccess(t *testing.T) {
	release := time.Date(1988, 8, 1, 0, 0, 0, 0, time.UTC)
	artistID := "c41d7f02-9b63-4a58-8e17-0d5b2a6c9e34"
	handler := newArtistHandler(artistRepoStub{
		artist: models.Artist{
			ID:        models.ArtistID(artistID),
			Name:      "Кино",
			Biography: "советская рок-группа",
		},
		tracks: []models.Track{{
			ID:         "3b8e5c90-77a1-4d2f-b6e4-12c9f0a5d738",
			Title:      "Группа крови",
			DurationMS: 214000,
			Artists:    []models.ArtistRef{{ID: models.ArtistID(artistID), Name: "Кино"}},
		}},
		albums: []models.Album{{
			ID:          "5e0a94c6-2f18-4b7d-a390-6c81e7f2b405",
			Title:       "Группа крови",
			ReleaseDate: &release,
			Artists:     []models.ArtistRef{{ID: models.ArtistID(artistID), Name: "Кино"}},
		}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/artists/"+artistID, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var body artistPageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("тело не разобралось: %v, тело: %s", err, w.Body.String())
	}
	if body.ID != artistID || body.Name != "Кино" || body.Biography != "советская рок-группа" {
		t.Fatalf("карточка исполнителя: %+v", body)
	}
	if len(body.Tracks) != 1 || body.Tracks[0].Title != "Группа крови" {
		t.Fatalf("треки: %+v", body.Tracks)
	}
	if len(body.Albums) != 1 || body.Albums[0].Title != "Группа крови" {
		t.Fatalf("альбомы: %+v", body.Albums)
	}
	if body.Albums[0].ReleaseDate == nil || *body.Albums[0].ReleaseDate != "1988-08-01" {
		t.Fatalf("release_date = %v", body.Albums[0].ReleaseDate)
	}
}

func TestArtistPageNotFound(t *testing.T) {
	handler := newArtistHandler(artistRepoStub{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/artists/c41d7f02-9b63-4a58-8e17-0d5b2a6c9e34", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
	if got := decodeError(t, w); got.Error.Code != models.CodeNotFound {
		t.Errorf("code = %q, ожидался %q", got.Error.Code, models.CodeNotFound)
	}
}

func TestArtistPageInvalidID(t *testing.T) {
	handler := newArtistHandler(artistRepoStub{
		artist: models.Artist{ID: "c41d7f02-9b63-4a58-8e17-0d5b2a6c9e34", Name: "Кино"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/artists/not-a-uuid", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}
