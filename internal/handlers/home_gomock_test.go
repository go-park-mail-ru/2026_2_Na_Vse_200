package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/apimessage"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	repositorymocks "github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository/mocks"
	"go.uber.org/mock/gomock"
)

// TestHomeWithMocks проверяет ответ главной страницы с данными из каталога.
func TestHomeWithMocks(t *testing.T) {
	catalog := repositorymocks.NewMockCatalogRepositoryInterface(gomock.NewController(t))
	catalog.EXPECT().HomeTracks(gomock.Any(), _homeLimit).Return([]models.Track{{
		ID:         "track-1",
		Title:      "Песня",
		DurationMS: 187000,
		Artists:    []models.ArtistRef{{ID: "artist-1", Name: "Исполнитель"}},
	}}, nil)
	catalog.EXPECT().HomeArtists(gomock.Any(), _homeLimit).Return([]models.Artist{{
		ID: "artist-1", Name: "Исполнитель",
	}}, nil)
	catalog.EXPECT().HomeAlbums(gomock.Any(), _homeLimit).Return([]models.Album{{
		ID: "album-1", Title: "Альбом",
	}}, nil)

	w := getHome(t, newHomeHandler(catalog))
	if w.Code != http.StatusOK {
		t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var body homeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("тело не разобралось: %v", err)
	}
	if len(body.Tracks) != 1 {
		t.Fatalf("треков = %d, ожидался 1", len(body.Tracks))
	}
	if body.Tracks[0].ID != "track-1" || body.Tracks[0].DurationMS != 187000 {
		t.Errorf("неверные данные трека: %+v", body.Tracks[0])
	}
	if len(body.Tracks[0].Artists) != 1 {
		t.Fatalf("исполнителей у трека = %d, ожидался 1", len(body.Tracks[0].Artists))
	}
	if body.Tracks[0].Artists[0].Name != "Исполнитель" {
		t.Errorf("исполнитель трека = %q, ожидался %q", body.Tracks[0].Artists[0].Name, "Исполнитель")
	}
	if len(body.Artists) != 1 {
		t.Fatalf("исполнителей = %d, ожидался 1", len(body.Artists))
	}
	if body.Artists[0].ID != "artist-1" {
		t.Errorf("id исполнителя = %q, ожидался %q", body.Artists[0].ID, "artist-1")
	}
	if len(body.Albums) != 1 {
		t.Fatalf("альбомов = %d, ожидался 1", len(body.Albums))
	}
	if body.Albums[0].Title != "Альбом" {
		t.Errorf("название альбома = %q, ожидалось %q", body.Albums[0].Title, "Альбом")
	}
}

// TestHomeRepositoryErrorWithMocks проверяет ответ при ошибке каталога.
func TestHomeRepositoryErrorWithMocks(t *testing.T) {
	storageErr := errors.New("internal database address 10.0.0.5:5432")
	tests := []struct {
		name  string
		setup func(*repositorymocks.MockCatalogRepositoryInterface)
	}{
		{
			name: "треки",
			setup: func(catalog *repositorymocks.MockCatalogRepositoryInterface) {
				catalog.EXPECT().HomeTracks(gomock.Any(), _homeLimit).Return(nil, storageErr)
			},
		},
		{
			name: "исполнители",
			setup: func(catalog *repositorymocks.MockCatalogRepositoryInterface) {
				catalog.EXPECT().HomeTracks(gomock.Any(), _homeLimit).Return(nil, nil)
				catalog.EXPECT().HomeArtists(gomock.Any(), _homeLimit).Return(nil, storageErr)
			},
		},
		{
			name: "альбомы",
			setup: func(catalog *repositorymocks.MockCatalogRepositoryInterface) {
				catalog.EXPECT().HomeTracks(gomock.Any(), _homeLimit).Return(nil, nil)
				catalog.EXPECT().HomeArtists(gomock.Any(), _homeLimit).Return(nil, nil)
				catalog.EXPECT().HomeAlbums(gomock.Any(), _homeLimit).Return(nil, storageErr)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog := repositorymocks.NewMockCatalogRepositoryInterface(gomock.NewController(t))
			tt.setup(catalog)

			w := getHome(t, newHomeHandler(catalog))
			if w.Code != http.StatusInternalServerError {
				t.Fatalf("статус = %d, ожидался %d", w.Code, http.StatusInternalServerError)
			}
			if got := decodeError(t, w); got.Error.Code != apimessage.CodeInternal {
				t.Errorf("code = %q, ожидался %q", got.Error.Code, apimessage.CodeInternal)
			}
			if strings.Contains(w.Body.String(), storageErr.Error()) {
				t.Errorf("внутренняя ошибка попала в ответ: %s", w.Body.String())
			}
		})
	}
}
