package artist

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"2026_2_Na_Vse_200/internal/models"
	artistusecase "2026_2_Na_Vse_200/internal/usecase/artist"
)

type pagedArtistStub struct{ limit, offset int }

func (s *pagedArtistStub) Page(context.Context, models.ArtistID) (artistusecase.ArtistPage, error) {
	return artistusecase.ArtistPage{}, models.ErrArtistNotFound
}
func (s *pagedArtistStub) PageAt(_ context.Context, id models.ArtistID, limit, offset int) (artistusecase.ArtistPage, error) {
	s.limit, s.offset = limit, offset
	return artistusecase.ArtistPage{Artist: models.Artist{ID: id, Name: "Artist"}}, nil
}
func TestArtistPagination(t *testing.T) {
	for _, tc := range []struct {
		query                 string
		status, limit, offset int
	}{
		{"?limit=10&offset=20", 200, 10, 20},
		{"?offset=100", 200, 100, 100},
		{"?limit=101", 400, 0, 0},
		{"?offset=-1", 400, 0, 0},
		{"?limit=no", 400, 0, 0},
	} {
		t.Run(tc.query, func(t *testing.T) {
			stub := &pagedArtistStub{}
			mux := http.NewServeMux()
			mux.HandleFunc("GET /artists/{id}", NewArtistHandler(stub).Get)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest("GET", "/artists/77777777-0000-4000-8000-000000000001"+tc.query, nil))
			if w.Code != tc.status || stub.limit != tc.limit || stub.offset != tc.offset {
				t.Fatalf("status=%d stub=%+v body=%s", w.Code, stub, w.Body.String())
			}
		})
	}
}
