package media

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"2026_2_Na_Vse_200/internal/models"
)

type mediaStub struct {
	file models.MediaFile
	err  error
}

func (s mediaStub) GetByID(context.Context, models.MediaFileID) (models.MediaFile, error) {
	return s.file, s.err
}

func (s mediaStub) GetAudio(context.Context, models.TrackID) (models.MediaFile, error) {
	return s.file, s.err
}

type blobStub struct {
	path string
	err  error
}

func (s blobStub) Open(string) (*os.File, error) {
	if s.err != nil {
		return nil, s.err
	}
	return os.Open(s.path)
}

// Аудио отдаётся целиком или нужным диапазоном байтов.
func TestAudio(t *testing.T) {
	path := filepath.Join(t.TempDir(), "track.mp3")
	if err := os.WriteFile(path, []byte("0123456789"), 0600); err != nil {
		t.Fatal(err)
	}
	repo := mediaStub{file: models.MediaFile{StorageKey: "track.mp3", MIMEType: "audio/mpeg"}}
	h := NewMediaHandler(repo, blobStub{path: path}, repo)

	for _, tc := range []struct {
		name, span, body string
		status           int
	}{
		{"целиком", "", "0123456789", http.StatusOK},
		{"перемотка", "bytes=2-5", "2345", http.StatusPartialContent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/audio", nil)
			r.SetPathValue("id", "11111111-1111-4111-8111-111111111111")
			if tc.span != "" {
				r.Header.Set("Range", tc.span)
			}
			w := httptest.NewRecorder()
			h.Audio(w, r)
			if w.Code != tc.status || w.Body.String() != tc.body {
				t.Fatalf("ответ = %d %q, ожидался %d %q", w.Code, w.Body.String(), tc.status, tc.body)
			}
			if w.Header().Get("Content-Type") != "audio/mpeg" || w.Header().Get("Accept-Ranges") != "bytes" {
				t.Errorf("неверные заголовки аудио: %v", w.Header())
			}
			if tc.span != "" && w.Header().Get("Content-Range") != "bytes 2-5/10" {
				t.Errorf("Content-Range = %q", w.Header().Get("Content-Range"))
			}
		})
	}
}

// Отсутствующий файл даёт 404, ошибка хранилища — 500.
func TestMediaErrors(t *testing.T) {
	for _, tc := range []struct {
		name, id         string
		repoErr, fileErr error
		status           int
	}{
		{"неверный ID", "bad-id", nil, nil, 404},
		{"нет метаданных", "11111111-1111-4111-8111-111111111111", models.ErrMediaFileNotFound, nil, 404},
		{"ошибка БД", "11111111-1111-4111-8111-111111111111", errors.New("db unavailable"), nil, 500},
		{"нет файла", "11111111-1111-4111-8111-111111111111", nil, os.ErrNotExist, 404},
		{"нет доступа", "11111111-1111-4111-8111-111111111111", nil, os.ErrPermission, 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := NewMediaHandler(mediaStub{err: tc.repoErr}, blobStub{err: tc.fileErr})
			r := httptest.NewRequest(http.MethodGet, "/media", nil)
			r.SetPathValue("id", tc.id)
			w := httptest.NewRecorder()
			h.Get(w, r)
			if w.Code != tc.status {
				t.Fatalf("статус = %d, ожидался %d", w.Code, tc.status)
			}
		})
	}
}

// Без репозитория треков аудио недоступно.
func TestAudioWithoutTracks(t *testing.T) {
	h := NewMediaHandler(nil, nil)
	r := httptest.NewRequest(http.MethodGet, "/audio", nil)
	r.SetPathValue("id", "11111111-1111-4111-8111-111111111111")
	w := httptest.NewRecorder()
	h.Audio(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("статус = %d, ожидался 404", w.Code)
	}
}
