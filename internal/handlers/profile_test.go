package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/apimessage"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/auth"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository/memory"
)

type countingUserRepo struct {
	repository.UserRepositoryInterface
	getByIDCalls int
}

func (r *countingUserRepo) GetByID(ctx context.Context, id models.UserID) (models.User, error) {
	r.getByIDCalls++
	return r.UserRepositoryInterface.GetByID(ctx, id)
}

type profileFixture struct {
	handler http.Handler
	users   *countingUserRepo
	cookie  *http.Cookie
	user    userResponse
}

func newProfileFixture(t *testing.T) profileFixture {
	t.Helper()

	users := &countingUserRepo{UserRepositoryInterface: memory.NewUserRepo()}
	api := New(&testConfig, &Deps{
		Users:    users,
		Sessions: memory.NewSessionRepo(),
		Hasher:   auth.NewBcryptHasherWithCost(auth.MinCost),
	})
	handler := middleware.Chain(api.Routes(), middleware.WithJSONErrors)

	w := postJSON(t, handler, "/api/v1/auth/signup",
		`{"email":"andrey@example.com","password":"muzyka2026","display_name":"Андрей"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("регистрация: статус = %d, тело: %s", w.Code, w.Body.String())
	}

	var user userResponse
	if err := json.Unmarshal(w.Body.Bytes(), &user); err != nil {
		t.Fatalf("тело не разобралось: %v, тело: %s", err, w.Body.String())
	}

	users.getByIDCalls = 0

	return profileFixture{
		handler: handler,
		users:   users,
		cookie:  sessionCookie(t, w),
		user:    user,
	}
}

func TestProfileOwner(t *testing.T) {
	f := newProfileFixture(t)

	tests := []struct {
		name string
		id   string
	}{
		{name: "id как выдан", id: f.user.ID},
		{name: "id в верхнем регистре", id: strings.ToUpper(f.user.ID)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := requestWithCookie(t, f.handler, http.MethodGet, "/api/v1/users/"+tt.id, f.cookie)

			if w.Code != http.StatusOK {
				t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusOK, w.Body.String())
			}

			var got profileResponse
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatalf("тело не разобралось: %v, тело: %s", err, w.Body.String())
			}

			if got.ID != f.user.ID {
				t.Errorf("id = %q, ожидался %q", got.ID, f.user.ID)
			}

			if got.Email != "andrey@example.com" {
				t.Errorf("email = %q, ожидался %q", got.Email, "andrey@example.com")
			}

			if got.DisplayName != "Андрей" {
				t.Errorf("display_name = %q, ожидался %q", got.DisplayName, "Андрей")
			}

			if got.AvatarURL != nil {
				t.Errorf("avatar_url = %v, ожидался null", *got.AvatarURL)
			}

			if got.CreatedAt.IsZero() || time.Since(got.CreatedAt) > time.Minute {
				t.Errorf("created_at = %v, ожидалось время регистрации", got.CreatedAt)
			}

			for _, forbidden := range []string{"muzyka2026", "password", "$2a$"} {
				if strings.Contains(w.Body.String(), forbidden) {
					t.Errorf("в ответе встречается %q: %s", forbidden, w.Body.String())
				}
			}
		})
	}
}

func TestProfileForbiddenForOtherUser(t *testing.T) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)

	f := newProfileFixture(t)

	w := requestWithCookie(t, f.handler, http.MethodGet, "/api/v1/users/"+uuid.New().String(), f.cookie)

	if w.Code != http.StatusForbidden {
		t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusForbidden, w.Body.String())
	}

	if got := decodeError(t, w); got.Error.Code != apimessage.CodeForbidden {
		t.Errorf("code = %q, ожидался %q", got.Error.Code, apimessage.CodeForbidden)
	}

	if f.users.getByIDCalls != 0 {
		t.Errorf("GetByID вызван %d раз: чужие данные не должны читаться", f.users.getByIDCalls)
	}
}

func TestProfileNotFoundForInvalidID(t *testing.T) {
	f := newProfileFixture(t)

	tests := []struct {
		name string
		id   string
	}{
		{name: "не uuid", id: "42"},
		{name: "обрезанный uuid", id: f.user.ID[:len(f.user.ID)-1]},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := requestWithCookie(t, f.handler, http.MethodGet, "/api/v1/users/"+tt.id, f.cookie)

			if w.Code != http.StatusNotFound {
				t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusNotFound, w.Body.String())
			}

			if got := decodeError(t, w); got.Error.Code != apimessage.CodeNotFound {
				t.Errorf("code = %q, ожидался %q", got.Error.Code, apimessage.CodeNotFound)
			}
		})
	}
}

func TestProfileUnauthorized(t *testing.T) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)

	f := newProfileFixture(t)

	tests := []struct {
		name   string
		cookie *http.Cookie
	}{
		{name: "cookie не прислана", cookie: nil},
		{name: "сессия неизвестна", cookie: &http.Cookie{Name: _sessionCookieName, Value: string(auth.NewSessionID())}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := requestWithCookie(t, f.handler, http.MethodGet, "/api/v1/users/"+f.user.ID, tt.cookie)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusUnauthorized, w.Body.String())
			}

			if got := decodeError(t, w); got.Error.Code != apimessage.CodeUnauthorized {
				t.Errorf("code = %q, ожидался %q", got.Error.Code, apimessage.CodeUnauthorized)
			}

			if f.users.getByIDCalls != 0 {
				t.Errorf("GetByID вызван %d раз без сессии", f.users.getByIDCalls)
			}
		})
	}
}

func TestProfileUserGone(t *testing.T) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)

	sessions := memory.NewSessionRepo()
	handler := newHandlerWithSessions(sessions)

	orphan := models.Session{
		ID:        auth.NewSessionID(),
		UserID:    models.UserID(uuid.New().String()),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := sessions.Create(context.Background(), orphan); err != nil {
		t.Fatalf("подготовка сессии: %v", err)
	}

	cookie := &http.Cookie{Name: _sessionCookieName, Value: string(orphan.ID)}
	w := requestWithCookie(t, handler, http.MethodGet, "/api/v1/users/"+string(orphan.UserID), cookie)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusUnauthorized, w.Body.String())
	}
}

func TestProfileStorageFailure(t *testing.T) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)

	sessions := memory.NewSessionRepo()
	api := New(&testConfig, &Deps{
		Users:    brokenUserRepo{},
		Sessions: sessions,
		Hasher:   auth.NewBcryptHasherWithCost(auth.MinCost),
	})
	handler := middleware.Chain(api.Routes(), middleware.WithJSONErrors)

	session := models.Session{
		ID:        auth.NewSessionID(),
		UserID:    models.UserID(uuid.New().String()),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := sessions.Create(context.Background(), session); err != nil {
		t.Fatalf("подготовка сессии: %v", err)
	}

	cookie := &http.Cookie{Name: _sessionCookieName, Value: string(session.ID)}
	w := requestWithCookie(t, handler, http.MethodGet, "/api/v1/users/"+string(session.UserID), cookie)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusInternalServerError, w.Body.String())
	}

	if got := decodeError(t, w); got.Error.Code != apimessage.CodeInternal {
		t.Errorf("code = %q, ожидался %q", got.Error.Code, apimessage.CodeInternal)
	}
}

type brokenUserRepo struct {
	repository.UserRepositoryInterface
}

func (brokenUserRepo) GetByID(context.Context, models.UserID) (models.User, error) {
	return models.User{}, errStorageDown
}
