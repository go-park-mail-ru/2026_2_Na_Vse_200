package media

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"

	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/storage"
)

func TestAudioRangeThroughLoggingMiddleware(t *testing.T) {
	blobs, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	file := models.MediaFile{StorageKey: "track.mp3", MIMEType: "audio/mpeg"}
	if err := blobs.Save(file.StorageKey, strings.NewReader("0123456789")); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/tracks/{id}/audio", NewMediaHandler(nil, blobs, audioTrackStub{file}).Audio)
	var logs bytes.Buffer
	handler := middleware.Chain(mux, middleware.WithRequestID, middleware.WithLogging(zerolog.New(&logs), "test"))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/018f3b2a-4c5d-4e6f-8a9b-0c1d2e3f4a5b/audio", nil)
	req.Header.Set("Range", "bytes=2-4")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusPartialContent || w.Body.String() != "234" || w.Header().Get("Content-Range") != "bytes 2-4/10" {
		t.Fatalf("неверный частичный ответ: статус %d, тело %q, Content-Range %q", w.Code, w.Body.String(), w.Header().Get("Content-Range"))
	}
	var event struct {
		Status        int    `json:"status"`
		ResponseBytes int64  `json:"response_bytes"`
		RequestID     string `json:"request_id"`
	}
	if err := json.Unmarshal(logs.Bytes(), &event); err != nil {
		t.Fatal(err)
	}
	if event.Status != http.StatusPartialContent || event.ResponseBytes != 3 ||
		event.RequestID == "" || event.RequestID != w.Header().Get(middleware.HeaderRequestID) {
		t.Fatalf("неверный access-log: %+v", event)
	}
}

type audioTrackStub struct{ file models.MediaFile }

func (s audioTrackStub) GetAudio(context.Context, models.TrackID) (models.MediaFile, error) {
	return s.file, nil
}
