package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"2026_2_Na_Vse_200/internal/config"
	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/catalog"
)

type artistStub struct {
	err                  error
	id                   models.ArtistID
	limit, offset, calls int
}

func (s *artistStub) Artist(_ context.Context, id models.ArtistID, limit, offset int) (catalog.ArtistPage, error) {
	s.id, s.limit, s.offset = id, limit, offset
	s.calls++
	return catalog.ArtistPage{
		Artist: models.Artist{ID: id, Name: "Artist", Biography: "Biography"},
	}, s.err
}

func TestArtistRoute(t *testing.T) {
	for _, tc := range []struct {
		name, query                  string
		err                          error
		status, calls, limit, offset int
	}{
		{"default", "", nil, 200, 1, 100, 0},
		{"pagination", "?limit=10&offset=20", nil, 200, 1, 10, 20},
		{"missing", "", models.ErrArtistNotFound, 404, 1, 100, 0},
		{"failure", "", errors.New("database failure"), 500, 1, 100, 0},
		{"negative offset", "?offset=-1", nil, 400, 0, 0, 0},
		{"large limit", "?limit=101", nil, 400, 0, 0, 0},
		{"invalid limit", "?limit=no", nil, 400, 0, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &artistStub{err: tc.err}
			h := New(&config.Config{}, &Deps{Artist: stub}).Routes()
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/artists/artist-id"+tc.query, nil))
			if w.Code != tc.status || stub.calls != tc.calls || stub.limit != tc.limit || stub.offset != tc.offset {
				t.Fatalf("status=%d stub=%+v body=%s", w.Code, stub, w.Body.String())
			}
			if tc.status != 200 {
				return
			}
			var body struct {
				ID, Name, Biography string
				Tracks, Albums      []json.RawMessage
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.ID != "artist-id" || body.Name != "Artist" || body.Biography != "Biography" || body.Tracks == nil || body.Albums == nil {
				t.Fatalf("unexpected body: %s", w.Body.String())
			}
		})
	}
}
