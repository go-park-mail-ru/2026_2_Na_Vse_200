package favorites

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"2026_2_Na_Vse_200/internal/models"
)

type repoStub struct {
	ids     []string
	err     error
	user    models.UserID
	track   models.TrackID
	enabled bool
}

func (s *repoStub) FavoriteIDs(_ context.Context, user models.UserID) ([]string, error) {
	s.user = user
	return s.ids, s.err
}

func (s *repoStub) SetFavorite(_ context.Context, user models.UserID, track models.TrackID, enabled bool) error {
	s.user, s.track, s.enabled = user, track, enabled
	return s.err
}

// withSession передаёт тестовую сессию через middleware авторизации.
func withSession(h http.HandlerFunc) http.Handler {
	return middleware.WithAuth("session_id", func(context.Context, models.SessionID) (models.Session, error) {
		return models.Session{UserID: "user-1"}, nil
	})(h)
}

func TestList(t *testing.T) {
	repo := &repoStub{ids: []string{"track-1", "track-2"}}
	h := NewFavoritesHandler(repo)
	r := httptest.NewRequest(http.MethodGet, "/favorites", nil)
	r.AddCookie(&http.Cookie{Name: "session_id", Value: "session-1"})
	w := httptest.NewRecorder()
	withSession(h.List).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("статус = %d, ожидался 200", w.Code)
	}
	var body struct {
		TrackIDs []string `json:"track_ids"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.TrackIDs) != 2 || body.TrackIDs[0] != "track-1" || body.TrackIDs[1] != "track-2" || repo.user != "user-1" {
		t.Fatalf("неверный список избранного: %+v, пользователь %q", body, repo.user)
	}
}

// TestSet проверяет добавление, удаление и ошибки избранного.
func TestSet(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	for _, tc := range []struct {
		name, method, id string
		err              error
		status           int
	}{
		{"добавить", http.MethodPut, id, nil, 204},
		{"удалить", http.MethodDelete, id, nil, 204},
		{"неверный ID", http.MethodPut, "bad-id", nil, 404},
		{"нет трека", http.MethodPut, id, models.ErrTrackNotFound, 404},
		{"ошибка БД", http.MethodPut, id, errors.New("db unavailable"), 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &repoStub{err: tc.err}
			h := NewFavoritesHandler(repo)
			r := httptest.NewRequest(tc.method, "/favorite", nil)
			r.SetPathValue("id", tc.id)
			r.AddCookie(&http.Cookie{Name: "session_id", Value: "session-1"})
			w := httptest.NewRecorder()
			withSession(h.Set).ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("статус = %d, ожидался %d", w.Code, tc.status)
			}
			if tc.status == 204 && (repo.user != "user-1" || repo.track != id || repo.enabled != (tc.method == http.MethodPut)) {
				t.Fatalf("неверные аргументы репозитория: %+v", repo)
			}
		})
	}
}

func TestFavoritesWithoutSession(t *testing.T) {
	h := NewFavoritesHandler(nil)
	for _, handler := range []http.HandlerFunc{h.List, h.Set} {
		w := httptest.NewRecorder()
		handler(w, httptest.NewRequest(http.MethodGet, "/favorites", nil))
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("статус = %d, ожидался 401", w.Code)
		}
	}
}

func TestListError(t *testing.T) {
	h := NewFavoritesHandler(&repoStub{err: errors.New("db unavailable")})
	r := httptest.NewRequest(http.MethodGet, "/favorites", nil)
	r.AddCookie(&http.Cookie{Name: "session_id", Value: "session-1"})
	w := httptest.NewRecorder()
	withSession(h.List).ServeHTTP(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("статус = %d, ожидался 500", w.Code)
	}
}
