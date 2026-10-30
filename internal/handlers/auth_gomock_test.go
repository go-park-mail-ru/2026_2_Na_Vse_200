package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/apimessage"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/auth"
	authmocks "github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/auth/mocks"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository"
	repositorymocks "github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository/mocks"
	"go.uber.org/mock/gomock"
)

// newMockAuthHandler собирает обработчик авторизации с моками зависимостей.
func newMockAuthHandler(
	users repository.UserRepositoryInterface,
	sessions repository.SessionRepositoryInterface,
	hasher auth.Hasher,
) http.Handler {
	api := New(&testConfig, &Deps{Users: users, Sessions: sessions, Hasher: hasher})
	return middleware.Chain(api.Routes(), middleware.WithJSONErrors)
}

// TestSignupWithMocks проверяет сохранение хеша и создание сессии при регистрации.
func TestSignupWithMocks(t *testing.T) {
	ctrl := gomock.NewController(t)
	users := repositorymocks.NewMockUserRepositoryInterface(ctrl)
	sessions := repositorymocks.NewMockSessionRepositoryInterface(ctrl)
	hasher := authmocks.NewMockHasher(ctrl)
	stored := models.User{
		ID:           "user-1",
		Email:        "andrey@example.com",
		DisplayName:  "Андрей",
		PasswordHash: "stored-hash",
	}
	var savedSession models.Session

	hasher.EXPECT().Hash("muzyka2026").Return("stored-hash", nil)
	users.EXPECT().Create(gomock.Any(), models.User{
		Email:        "andrey@example.com",
		DisplayName:  "Андрей",
		PasswordHash: "stored-hash",
	}).Return(stored, nil)
	sessions.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, session models.Session) error {
			savedSession = session
			return nil
		},
	)

	w := postJSON(t, newMockAuthHandler(users, sessions, hasher), "/api/v1/auth/signup",
		`{"email":"Andrey@Example.com","password":"muzyka2026","display_name":"Андрей"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusCreated, w.Body.String())
	}
	if savedSession.UserID != stored.ID {
		t.Errorf("user_id сессии = %q, ожидался %q", savedSession.UserID, stored.ID)
	}
	if savedSession.ID == "" {
		t.Fatal("у сохранённой сессии пустой id")
	}
	if !savedSession.ExpiresAt.After(time.Now()) {
		t.Errorf("сессия уже истекла: %s", savedSession.ExpiresAt)
	}
	if cookie := sessionCookie(t, w); cookie.Value != string(savedSession.ID) {
		t.Errorf("cookie = %q, ожидался ID сохранённой сессии %q", cookie.Value, savedSession.ID)
	}

	var body userResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("тело не разобралось: %v", err)
	}
	if body.ID != string(stored.ID) {
		t.Errorf("id в ответе = %q, ожидался %q", body.ID, stored.ID)
	}
	if body.Email != stored.Email {
		t.Errorf("email в ответе = %q, ожидался %q", body.Email, stored.Email)
	}
	if body.DisplayName != stored.DisplayName {
		t.Errorf("display_name в ответе = %q, ожидался %q", body.DisplayName, stored.DisplayName)
	}
}

// TestLoginWrongPasswordWithMocks проверяет отказ во входе без создания сессии.
func TestLoginWrongPasswordWithMocks(t *testing.T) {
	ctrl := gomock.NewController(t)
	users := repositorymocks.NewMockUserRepositoryInterface(ctrl)
	sessions := repositorymocks.NewMockSessionRepositoryInterface(ctrl)
	hasher := authmocks.NewMockHasher(ctrl)
	users.EXPECT().GetByEmail(gomock.Any(), "andrey@example.com").Return(models.User{
		ID: "user-1", PasswordHash: "stored-hash",
	}, nil)
	hasher.EXPECT().Verify("wrong-password", "stored-hash").Return(false, nil)
	w := postJSON(t, newMockAuthHandler(users, sessions, hasher), "/api/v1/auth/login",
		`{"email":"Andrey@Example.com","password":"wrong-password"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("статус = %d, ожидался %d", w.Code, http.StatusUnauthorized)
	}
	if got := decodeError(t, w); got.Error.Code != apimessage.CodeInvalidCredentials {
		t.Errorf("code = %q, ожидался %q", got.Error.Code, apimessage.CodeInvalidCredentials)
	}
	if len(w.Result().Cookies()) != 0 {
		t.Errorf("при неверном пароле выставлена cookie: %v", w.Result().Cookies())
	}
}
