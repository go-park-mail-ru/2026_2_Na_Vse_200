package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"2026_2_Na_Vse_200/internal/models"
)

type sessionRepoStub struct {
	session models.Session
	err     error
}

func (s sessionRepoStub) Create(context.Context, models.Session) error { return nil }
func (s sessionRepoStub) Delete(context.Context, models.SessionID) error {
	return nil
}
func (s sessionRepoStub) GetByID(context.Context, models.SessionID) (models.Session, error) {
	return s.session, s.err
}

func TestWithAuth(t *testing.T) {
	valid := models.Session{
		ID:        "11111111-1111-1111-1111-111111111111",
		UserID:    "22222222-2222-2222-2222-222222222222",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	tests := []struct {
		name       string
		cookie     *http.Cookie
		repo       sessionRepoStub
		wantStatus int
		wantUser   models.UserID
	}{
		{
			name:       "нет cookie",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "сессия неизвестна",
			cookie:     &http.Cookie{Name: "session_id", Value: "missing"},
			repo:       sessionRepoStub{err: models.ErrSessionNotFound},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "сессия истекла",
			cookie:     &http.Cookie{Name: "session_id", Value: "expired"},
			repo:       sessionRepoStub{err: models.ErrSessionExpired},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "ошибка хранилища",
			cookie:     &http.Cookie{Name: "session_id", Value: "broken"},
			repo:       sessionRepoStub{err: errors.New("db down")},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "успех",
			cookie:     &http.Cookie{Name: "session_id", Value: string(valid.ID)},
			repo:       sessionRepoStub{session: valid},
			wantStatus: http.StatusOK,
			wantUser:   valid.UserID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotUser models.UserID
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				session, ok := SessionFromContext(r.Context())
				if !ok {
					t.Fatal("сессия не попала в контекст")
				}
				gotUser = session.UserID
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			w := httptest.NewRecorder()
			WithAuth("session_id", tt.repo)(next).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantStatus == http.StatusOK && gotUser != tt.wantUser {
				t.Fatalf("user id = %q, ожидался %q", gotUser, tt.wantUser)
			}
		})
	}
}
