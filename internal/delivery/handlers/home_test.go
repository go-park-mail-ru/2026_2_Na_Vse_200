package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
)

func newHomeHandler(catalog ports.CatalogRepository) http.Handler {
	api := newRepositoryTestAPI(&testConfig, &repositoryTestDeps{Catalog: catalog})
	return middleware.Chain(api.Routes(), middleware.WithJSONErrors)
}

func getHome(t *testing.T, handler http.Handler) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/home", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	return w
}

func TestHomeSuccess(t *testing.T) {
	w := getHome(t, newHomeHandler(homeCatalogFixture{}))

	if w.Code != http.StatusOK {
		t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusOK, w.Body.String())
	}

	if got := w.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Errorf("Content-Type = %q, ожидался JSON", got)
	}

	var home homeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &home); err != nil {
		t.Fatalf("тело не разобралось: %v, тело: %s", err, w.Body.String())
	}

	if len(home.Tracks) == 0 {
		t.Error("в ответе нет треков")
	}

	if len(home.Artists) == 0 {
		t.Error("в ответе нет исполнителей")
	}

	if len(home.Albums) == 0 {
		t.Error("в ответе нет альбомов")
	}
}

func TestHomeTrackCardIsComplete(t *testing.T) {
	w := getHome(t, newHomeHandler(homeCatalogFixture{}))

	var home homeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &home); err != nil {
		t.Fatalf("тело не разобралось: %v", err)
	}

	if len(home.Tracks) == 0 {
		t.Fatal("в ответе нет треков")
	}

	track := home.Tracks[0]

	if track.ID == "" {
		t.Error("у трека нет id")
	}

	if track.Title == "" {
		t.Error("у трека нет названия")
	}

	if track.DurationMS <= 0 {
		t.Errorf("длительность = %d, ожидалась положительная", track.DurationMS)
	}

	if len(track.Artists) == 0 {
		t.Error("у трека нет исполнителей")
	}
}

type emptyCatalogRepo struct{}

func (emptyCatalogRepo) HomeTracks(context.Context, int) ([]models.Track, error) {
	return nil, nil
}

func (emptyCatalogRepo) HomeArtists(context.Context, int) ([]models.Artist, error) {
	return nil, nil
}

func (emptyCatalogRepo) HomeAlbums(context.Context, int) ([]models.Album, error) {
	return nil, nil
}

// Пустой список должен передаваться как [], а не null.
func TestHomeEmptyCatalogReturnsArrays(t *testing.T) {
	w := getHome(t, newHomeHandler(emptyCatalogRepo{}))

	if w.Code != http.StatusOK {
		t.Fatalf("статус = %d, ожидался %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if strings.Contains(body, "null") {
		t.Errorf("в ответе есть null вместо пустого массива: %s", body)
	}

	for _, want := range []string{`"tracks":[]`, `"artists":[]`, `"albums":[]`} {
		if !strings.Contains(body, want) {
			t.Errorf("в ответе нет %s, получено: %s", want, body)
		}
	}
}

type brokenCatalogRepo struct{}

func (brokenCatalogRepo) HomeTracks(context.Context, int) ([]models.Track, error) {
	return nil, errStorageDown
}

func (brokenCatalogRepo) HomeArtists(context.Context, int) ([]models.Artist, error) {
	return nil, errStorageDown
}

func (brokenCatalogRepo) HomeAlbums(context.Context, int) ([]models.Album, error) {
	return nil, errStorageDown
}

func TestHomeStorageFailure(t *testing.T) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)

	w := getHome(t, newHomeHandler(brokenCatalogRepo{}))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusInternalServerError, w.Body.String())
	}

	if got := decodeError(t, w); got.Error.Code != models.CodeInternal {
		t.Errorf("code = %q, ожидался %q", got.Error.Code, models.CodeInternal)
	}

	if strings.Contains(w.Body.String(), "5432") {
		t.Errorf("детали ошибки хранилища ушли клиенту: %s", w.Body.String())
	}
}

func TestHomeMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/home", nil)
	w := httptest.NewRecorder()

	newHomeHandler(homeCatalogFixture{}).ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("статус = %d, ожидался %d", w.Code, http.StatusMethodNotAllowed)
	}
}
