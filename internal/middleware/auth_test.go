package middleware

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/apimessage"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository/memory"
)

const _testCookieName = "session_id"

var errSessionStorageDown = errors.New("хранилище сессий недоступно")

type brokenSessionRepo struct {
	repository.SessionRepositoryInterface
}

func (brokenSessionRepo) GetByID(context.Context, models.SessionID) (models.Session, error) {
	return models.Session{}, errSessionStorageDown
}

func TestWithAuth(t *testing.T) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)

	sessions := memory.NewSessionRepo()
	alive := models.Session{ID: "alive", UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour)}
	expired := models.Session{ID: "expired", UserID: "user-2", ExpiresAt: time.Now().Add(-time.Minute)}
	for _, session := range []models.Session{alive, expired} {
		if err := sessions.Create(context.Background(), session); err != nil {
			t.Fatalf("подготовка сессии: %v", err)
		}
	}

	tests := []struct {
		name        string
		sessions    repository.SessionRepositoryInterface
		cookie      *http.Cookie
		wantCode    int
		wantErrCode string
		wantNext    bool
		wantUserID  models.UserID
	}{
		{
			name:        "без cookie",
			sessions:    sessions,
			wantCode:    http.StatusUnauthorized,
			wantErrCode: apimessage.CodeUnauthorized,
		},
		{
			name:        "сессия неизвестна",
			sessions:    sessions,
			cookie:      &http.Cookie{Name: _testCookieName, Value: "unknown"},
			wantCode:    http.StatusUnauthorized,
			wantErrCode: apimessage.CodeUnauthorized,
		},
		{
			name:        "сессия истекла",
			sessions:    sessions,
			cookie:      &http.Cookie{Name: _testCookieName, Value: string(expired.ID)},
			wantCode:    http.StatusUnauthorized,
			wantErrCode: apimessage.CodeUnauthorized,
		},
		{
			name:        "хранилище недоступно",
			sessions:    brokenSessionRepo{},
			cookie:      &http.Cookie{Name: _testCookieName, Value: string(alive.ID)},
			wantCode:    http.StatusInternalServerError,
			wantErrCode: apimessage.CodeInternal,
		},
		{
			name:       "сессия действует",
			sessions:   sessions,
			cookie:     &http.Cookie{Name: _testCookieName, Value: string(alive.ID)},
			wantCode:   http.StatusOK,
			wantNext:   true,
			wantUserID: alive.UserID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				nextCalled bool
				gotSession models.Session
				gotOK      bool
			)
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				gotSession, gotOK = SessionFromContext(r.Context())
			})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/users/1", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			w := httptest.NewRecorder()

			WithAuth(_testCookieName, tt.sessions)(next).ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, tt.wantCode, w.Body.String())
			}

			if nextCalled != tt.wantNext {
				t.Fatalf("следующий обработчик вызван = %v, ожидалось %v", nextCalled, tt.wantNext)
			}

			if !tt.wantNext {
				if got := decodeError(t, w); got.Error.Code != tt.wantErrCode {
					t.Errorf("code = %q, ожидался %q", got.Error.Code, tt.wantErrCode)
				}
				return
			}

			if !gotOK {
				t.Fatal("сессии нет в контексте")
			}

			if gotSession.UserID != tt.wantUserID {
				t.Errorf("UserID = %q, ожидался %q", gotSession.UserID, tt.wantUserID)
			}
		})
	}
}

func TestSessionFromContextEmpty(t *testing.T) {
	if _, ok := SessionFromContext(context.Background()); ok {
		t.Error("пустой контекст не должен содержать сессию")
	}
}
