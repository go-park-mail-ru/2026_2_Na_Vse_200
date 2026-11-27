package handlers

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"2026_2_Na_Vse_200/internal/models"
	authusecase "2026_2_Na_Vse_200/internal/usecase/auth"
	authmocks "2026_2_Na_Vse_200/internal/usecase/auth/mocks"
	"2026_2_Na_Vse_200/internal/usecase/ports"
	portmocks "2026_2_Na_Vse_200/internal/usecase/ports/mocks"
	"go.uber.org/mock/gomock"
)

// newMockAuthHandler собирает обработчик авторизации с моками зависимостей.
func newMockAuthHandler(
	users ports.UserRepository,
	sessions authusecase.Sessions,
	hasher ports.PasswordHasher,
) http.Handler {
	api := newRepositoryTestAPI(&testConfig, &repositoryTestDeps{Users: users, Sessions: sessions, Hasher: hasher})
	return middleware.Chain(api.Routes(), middleware.WithJSONErrors)
}

// TestSignupWithMocks проверяет сохранение хеша и создание сессии при регистрации.
func TestSignupWithMocks(t *testing.T) {
	ctrl := gomock.NewController(t)
	users := portmocks.NewMockUserRepository(ctrl)
	sessions := authmocks.NewMockSessions(ctrl)
	hasher := portmocks.NewMockPasswordHasher(ctrl)
	stored := models.User{
		ID:                "user-1",
		Email:             "andrey@example.com",
		DisplayName:       "Андрей",
		PasswordHash:      "stored-hash",
		CredentialVersion: 1,
	}
	session := models.Session{ID: "test-session", UserID: stored.ID, ExpiresAt: time.Now().Add(time.Hour)}

	hasher.EXPECT().Hash("muzyka2026").Return("stored-hash", nil)
	users.EXPECT().Create(gomock.Any(), models.User{
		Email:        "andrey@example.com",
		DisplayName:  "Андрей",
		PasswordHash: "stored-hash",
	}).Return(stored, nil)
	sessions.EXPECT().CreateSession(gomock.Any(), stored.ID, stored.CredentialVersion).Return(session, nil)

	w := postJSON(t, newMockAuthHandler(users, sessions, hasher), "/api/v1/auth/signup",
		`{"email":"Andrey@Example.com","password":"muzyka2026","display_name":"Андрей"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusCreated, w.Body.String())
	}
	if cookie := sessionCookie(t, w); cookie.Value != string(session.ID) {
		t.Errorf("cookie = %q, ожидался ID сессии от Auth %q", cookie.Value, session.ID)
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
	users := portmocks.NewMockUserRepository(ctrl)
	sessions := authmocks.NewMockSessions(ctrl)
	hasher := portmocks.NewMockPasswordHasher(ctrl)
	users.EXPECT().GetByEmail(gomock.Any(), "andrey@example.com").Return(models.User{
		ID: "user-1", PasswordHash: "stored-hash",
	}, nil)
	hasher.EXPECT().Verify("wrong-password", "stored-hash").Return(false, nil)
	w := postJSON(t, newMockAuthHandler(users, sessions, hasher), "/api/v1/auth/login",
		`{"email":"Andrey@Example.com","password":"wrong-password"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("статус = %d, ожидался %d", w.Code, http.StatusUnauthorized)
	}
	if got := decodeError(t, w); got.Error.Code != models.CodeInvalidCredentials {
		t.Errorf("code = %q, ожидался %q", got.Error.Code, models.CodeInvalidCredentials)
	}
	if len(w.Result().Cookies()) != 0 {
		t.Errorf("при неверном пароле выставлена cookie: %v", w.Result().Cookies())
	}
}
