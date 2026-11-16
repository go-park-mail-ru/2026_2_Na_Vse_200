package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"2026_2_Na_Vse_200/internal/models"

	"github.com/rs/zerolog"
)

func TestWithAuth(t *testing.T) {
	valid := models.Session{
		ID:        "11111111-1111-1111-1111-111111111111",
		UserID:    "22222222-2222-2222-2222-222222222222",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	tests := []struct {
		name       string
		cookie     *http.Cookie
		lookup     SessionByIDFunc
		wantStatus int
		wantUser   models.UserID
	}{
		{
			name:       "нет cookie",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "сессия неизвестна",
			cookie: &http.Cookie{Name: "session_id", Value: "missing"},
			lookup: func(context.Context, models.SessionID) (models.Session, error) {
				return models.Session{}, models.ErrSessionNotFound
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "сессия истекла",
			cookie: &http.Cookie{Name: "session_id", Value: "expired"},
			lookup: func(context.Context, models.SessionID) (models.Session, error) {
				return models.Session{}, models.ErrSessionExpired
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "ошибка хранилища",
			cookie: &http.Cookie{Name: "session_id", Value: "broken"},
			lookup: func(context.Context, models.SessionID) (models.Session, error) {
				return models.Session{}, errors.New("db down")
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "успех",
			cookie: &http.Cookie{Name: "session_id", Value: string(valid.ID)},
			lookup: func(context.Context, models.SessionID) (models.Session, error) {
				return valid, nil
			},
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
			WithAuth("session_id", tt.lookup)(next).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantStatus == http.StatusOK && gotUser != tt.wantUser {
				t.Fatalf("user id = %q, ожидался %q", gotUser, tt.wantUser)
			}
		})
	}
}

func TestWithAuthScopesUserLogger(t *testing.T) {
	logger, buf := testLogger()
	lookup := func(_ context.Context, id models.SessionID) (models.Session, error) {
		switch id {
		case "private-session-a":
			return models.Session{ID: id, UserID: "user-a"}, nil
		case "private-session-b":
			return models.Session{ID: id, UserID: "user-b"}, nil
		default:
			return models.Session{}, models.ErrSessionNotFound
		}
	}

	for _, tt := range []struct {
		cookie string
		user   models.UserID
		status int
	}{
		{"private-session-a", "user-a", http.StatusNoContent},
		{"private-session-b", "user-b", http.StatusNoContent},
		{"private-session-invalid", "", http.StatusUnauthorized},
	} {
		t.Run(tt.cookie, func(t *testing.T) {
			buf.Reset()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			called := false
			handler := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				session, ok := SessionFromContext(r.Context())
				if !ok || session.UserID != tt.user {
					t.Fatalf("сессия = %+v, ожидался %q", session, tt.user)
				}
				cancel()
				if !errors.Is(r.Context().Err(), context.Canceled) {
					t.Fatal("обработчик потерял сигнал отмены запроса")
				}
				zerolog.Ctx(r.Context()).Info().Msg("protected handler")
				w.WriteHeader(http.StatusNoContent)
			}), WithRequestID, WithLogging(logger, "test"), WithAuth("session_id", lookup))

			req := httptest.NewRequest(http.MethodGet, "/protected", nil).WithContext(ctx)
			req.AddCookie(&http.Cookie{Name: "session_id", Value: tt.cookie})
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != tt.status || called != (tt.user != "") {
				t.Fatalf("статус = %d, обработчик вызван = %t", w.Code, called)
			}

			events := readRegressionEvents(t, buf)
			if tt.user != "" {
				if len(events) != 2 || events[0]["user_id"] != string(tt.user) ||
					events[0]["request_id"] != w.Header().Get(HeaderRequestID) {
					t.Fatalf("неверный лог защищённого обработчика: %v", events)
				}
			} else if len(events) != 1 {
				t.Fatalf("ожидалось одно событие access-log, получено %v", events)
			}
			if _, ok := events[len(events)-1]["user_id"]; ok {
				t.Fatal("пользователь появился во внешнем access-log")
			}
			if strings.Contains(buf.String(), tt.cookie) {
				t.Fatal("лог содержит идентификатор сессии")
			}
		})
	}
}
