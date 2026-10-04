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
	"time"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/apimessage"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/auth"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository/memory"
)

// newHomeHandler собирает обработчики с настоящим каталогом в памяти.
func newHomeHandler(catalog repository.CatalogRepositoryInterface) http.Handler {
	api := New(&testConfig, &Deps{
		Users:    memory.NewUserRepo(),
		Sessions: memory.NewSessionRepo(),
		Catalog:  catalog,
		Hasher:   auth.NewBcryptHasherWithCost(auth.MinCost),
	})
	return middleware.Chain(api.Routes(), middleware.WithJSONErrors)
}

// getHome запрашивает главную и возвращает ответ.
func getHome(t *testing.T, handler http.Handler) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/home", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	return w
}

func TestHomeSuccess(t *testing.T) {
	w := getHome(t, newHomeHandler(memory.NewCatalogRepo()))

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

// Карточка трека должна приходить собранной: без исполнителя и длительности
// фронтенду нечего показывать в списке.
func TestHomeTrackCardIsComplete(t *testing.T) {
	w := getHome(t, newHomeHandler(memory.NewCatalogRepo()))

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

// emptyCatalogRepo отдаёт пустой каталог, причём срезами nil.
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

// Пустой раздел обязан приходить как [], а не null: контракт требует массив,
// а json.Marshal превращает nil-срез именно в null.
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

// brokenCatalogRepo изображает недоступное хранилище каталога.
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

	if got := decodeError(t, w); got.Error.Code != apimessage.CodeInternal {
		t.Errorf("code = %q, ожидался %q", got.Error.Code, apimessage.CodeInternal)
	}

	if strings.Contains(w.Body.String(), "5432") {
		t.Errorf("детали ошибки хранилища ушли клиенту: %s", w.Body.String())
	}
}

// Главная отдаётся только по GET.
func TestHomeMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/home", nil)
	w := httptest.NewRecorder()

	newHomeHandler(memory.NewCatalogRepo()).ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("статус = %d, ожидался %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestOptionalString(t *testing.T) {
	if got := optionalString(""); got != nil {
		t.Errorf("для пустой строки вернулось %q, ожидался nil", *got)
	}

	const url = "https://cdn.example.com/covers/1001.jpg"
	got := optionalString(url)
	if got == nil {
		t.Fatal("для непустой строки вернулся nil")
	}

	if *got != url {
		t.Errorf("значение = %q, ожидалось %q", *got, url)
	}
}

func TestFormatReleaseDate(t *testing.T) {
	if got := formatReleaseDate(nil); got != nil {
		t.Errorf("для неизвестной даты вернулось %q, ожидался nil", *got)
	}

	// Время берём не в UTC: формат обязан приводить дату к UTC, иначе на стенде
	// в другом часовом поясе дата уехала бы на сутки.
	moscow := time.FixedZone("MSK", 3*60*60)
	date := time.Date(2024, 8, 1, 1, 30, 0, 0, moscow)

	got := formatReleaseDate(&date)
	if got == nil {
		t.Fatal("для известной даты вернулся nil")
	}

	if *got != "2024-07-31" {
		t.Errorf("дата = %q, ожидалась %q", *got, "2024-07-31")
	}
}
