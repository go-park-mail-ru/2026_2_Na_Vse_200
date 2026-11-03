package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
	"uuid"

	"2026_2_Na_Vse_200/internal/auth"
	"2026_2_Na_Vse_200/internal/config"
	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
)

var testConfig = config.Config{SessionTTL: time.Hour}

func newAuthHandler() http.Handler {
	api := newRepositoryTestAPI(&testConfig, &repositoryTestDeps{
		Users:    &authUsersFixture{},
		Sessions: &authSessionsFixture{},
		Hasher:   auth.NewBcryptHasherWithCost(auth.MinCost),
	})
	return middleware.Chain(api.Routes(), middleware.WithJSONErrors)
}

func sessionCookie(t *testing.T, w *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()

	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == _sessionCookieName {
			return cookie
		}
	}

	t.Fatalf("в ответе нет cookie %q", _sessionCookieName)
	return nil
}

func postJSON(t *testing.T, handler http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	return w
}

func TestSignupSuccess(t *testing.T) {
	handler := newAuthHandler()

	w := postJSON(t, handler, "/api/v1/auth/signup",
		`{"email":"Andrey@Example.com","password":"muzyka2026","display_name":"Андрей"}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var got userResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("тело не разобралось: %v, тело: %s", err, w.Body.String())
	}

	if got.ID == "" {
		t.Error("в ответе нет id")
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

	cookie := sessionCookie(t, w)
	if cookie.Value == "" {
		t.Error("cookie сессии пустая")
	}

	if !cookie.HttpOnly {
		t.Error("cookie сессии без HttpOnly: её прочитает JavaScript")
	}

	if cookie.Path != "/" {
		t.Errorf("Path = %q, ожидался %q", cookie.Path, "/")
	}
}

func TestSignupResponseHasNoPassword(t *testing.T) {
	handler := newAuthHandler()

	w := postJSON(t, handler, "/api/v1/auth/signup",
		`{"email":"andrey@example.com","password":"muzyka2026","display_name":"Андрей"}`)

	body := w.Body.String()
	for _, forbidden := range []string{"muzyka2026", "password", "sha256"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("в ответе встречается %q: %s", forbidden, body)
		}
	}
}

func TestSignupErrors(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantCode   int
		wantErr    string
		wantFields []string
	}{
		{
			name:     "тело не разобралось",
			body:     `{"email": "andrey@example.com"`,
			wantCode: http.StatusBadRequest,
			wantErr:  models.CodeInvalidJSON,
		},
		{
			name:     "пустое тело",
			body:     ``,
			wantCode: http.StatusBadRequest,
			wantErr:  models.CodeInvalidJSON,
		},
		{
			name:       "нет обязательных полей",
			body:       `{}`,
			wantCode:   http.StatusBadRequest,
			wantErr:    models.CodeValidationFailed,
			wantFields: []string{"email", "password", "display_name"},
		},
		{
			name:       "короткий пароль",
			body:       `{"email":"andrey@example.com","password":"123","display_name":"Андрей"}`,
			wantCode:   http.StatusBadRequest,
			wantErr:    models.CodeValidationFailed,
			wantFields: []string{"password"},
		},
		{
			name:       "некорректный email",
			body:       `{"email":"без-собаки","password":"muzyka2026","display_name":"Андрей"}`,
			wantCode:   http.StatusBadRequest,
			wantErr:    models.CodeValidationFailed,
			wantFields: []string{"email"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := postJSON(t, newAuthHandler(), "/api/v1/auth/signup", tt.body)

			if w.Code != tt.wantCode {
				t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, tt.wantCode, w.Body.String())
			}

			got := decodeError(t, w)
			if got.Error.Code != tt.wantErr {
				t.Errorf("code = %q, ожидался %q", got.Error.Code, tt.wantErr)
			}
			for _, field := range tt.wantFields {
				if got.Error.Fields[field] == "" {
					t.Errorf("в fields нет поля %q, получено: %v", field, got.Error.Fields)
				}
			}
		})
	}
}

func TestSignupEmailTaken(t *testing.T) {
	handler := newAuthHandler()
	const body = `{"email":"andrey@example.com","password":"muzyka2026","display_name":"Андрей"}`

	if w := postJSON(t, handler, "/api/v1/auth/signup", body); w.Code != http.StatusCreated {
		t.Fatalf("первая регистрация: статус = %d, тело: %s", w.Code, w.Body.String())
	}

	w := postJSON(t, handler, "/api/v1/auth/signup", body)
	if w.Code != http.StatusConflict {
		t.Fatalf("вторая регистрация: статус = %d, ожидался %d", w.Code, http.StatusConflict)
	}

	got := decodeError(t, w)
	if got.Error.Code != models.CodeEmailTaken {
		t.Errorf("code = %q, ожидался %q", got.Error.Code, models.CodeEmailTaken)
	}
}

func TestSignupEmailCaseInsensitive(t *testing.T) {
	handler := newAuthHandler()

	if w := postJSON(t, handler, "/api/v1/auth/signup",
		`{"email":"andrey@example.com","password":"muzyka2026","display_name":"Андрей"}`); w.Code != http.StatusCreated {
		t.Fatalf("первая регистрация: статус = %d", w.Code)
	}

	w := postJSON(t, handler, "/api/v1/auth/signup",
		`{"email":"ANDREY@Example.COM","password":"muzyka2026","display_name":"Андрей"}`)
	if w.Code != http.StatusConflict {
		t.Errorf("статус = %d, ожидался %d: email в другом регистре — тот же аккаунт", w.Code, http.StatusConflict)
	}
}

type failingUserRepo struct {
	ports.UserRepository
}

var errStorageDown = errors.New("хранилище недоступно: connection refused to 10.0.0.5:5432")

func (failingUserRepo) Create(ctx context.Context, user models.User) (models.User, error) {
	return models.User{}, errStorageDown
}

func TestSignupStorageFailure(t *testing.T) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)

	api := newRepositoryTestAPI(&testConfig, &repositoryTestDeps{
		Users:    failingUserRepo{},
		Sessions: &authSessionsFixture{},
		Hasher:   auth.NewBcryptHasherWithCost(auth.MinCost),
	})
	handler := middleware.Chain(api.Routes(), middleware.WithJSONErrors)

	w := postJSON(t, handler, "/api/v1/auth/signup",
		`{"email":"andrey@example.com","password":"muzyka2026","display_name":"Андрей"}`)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("статус = %d, ожидался %d", w.Code, http.StatusInternalServerError)
	}

	got := decodeError(t, w)
	if got.Error.Code != models.CodeInternal {
		t.Errorf("code = %q, ожидался %q", got.Error.Code, models.CodeInternal)
	}

	if strings.Contains(w.Body.String(), "5432") {
		t.Errorf("детали ошибки хранилища ушли клиенту: %s", w.Body.String())
	}
}

type failingSessionRepo struct {
	ports.SessionRepository
}

func (failingSessionRepo) Create(ctx context.Context, session models.Session) error {
	return errStorageDown
}

func newHandlerWithSessions(sessions ports.SessionRepository) http.Handler {
	api := newRepositoryTestAPI(&testConfig, &repositoryTestDeps{
		Users:    &authUsersFixture{},
		Sessions: sessions,
		Hasher:   auth.NewBcryptHasherWithCost(auth.MinCost),
	})
	return middleware.Chain(api.Routes(), middleware.WithJSONErrors)
}

// Сбой создания сессии не отменяет уже созданный аккаунт.
func TestSignupSessionFailure(t *testing.T) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)

	users := &authUsersFixture{}
	api := newRepositoryTestAPI(&testConfig, &repositoryTestDeps{
		Users:    users,
		Sessions: failingSessionRepo{},
		Hasher:   auth.NewBcryptHasherWithCost(auth.MinCost),
	})
	handler := middleware.Chain(api.Routes(), middleware.WithJSONErrors)

	w := postJSON(t, handler, "/api/v1/auth/signup",
		`{"email":"andrey@example.com","password":"muzyka2026","display_name":"Андрей"}`)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusInternalServerError, w.Body.String())
	}

	if got := decodeError(t, w); got.Error.Code != models.CodeInternal {
		t.Errorf("code = %q, ожидался %q", got.Error.Code, models.CodeInternal)
	}

	if len(w.Result().Cookies()) != 0 {
		t.Error("выставлена cookie, хотя сессия не сохранилась")
	}

	if _, err := users.GetByEmail(context.Background(), "andrey@example.com"); err != nil {
		t.Errorf("аккаунт не найден после отказа: %v", err)
	}
}

func TestLoginSuccess(t *testing.T) {
	handler := newAuthHandler()

	if w := postJSON(t, handler, "/api/v1/auth/signup",
		`{"email":"andrey@example.com","password":"muzyka2026","display_name":"Андрей"}`); w.Code != http.StatusCreated {
		t.Fatalf("регистрация: статус = %d, тело: %s", w.Code, w.Body.String())
	}

	w := postJSON(t, handler, "/api/v1/auth/login",
		`{"email":"ANDREY@example.com","password":"muzyka2026"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var got userResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("тело не разобралось: %v, тело: %s", err, w.Body.String())
	}

	if got.Email != "andrey@example.com" {
		t.Errorf("email = %q, ожидался %q", got.Email, "andrey@example.com")
	}

	if cookie := sessionCookie(t, w); !cookie.HttpOnly {
		t.Error("cookie сессии без HttpOnly: её прочитает JavaScript")
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	handler := newAuthHandler()

	if w := postJSON(t, handler, "/api/v1/auth/signup",
		`{"email":"andrey@example.com","password":"muzyka2026","display_name":"Андрей"}`); w.Code != http.StatusCreated {
		t.Fatalf("регистрация: статус = %d, тело: %s", w.Code, w.Body.String())
	}

	wrongPassword := postJSON(t, handler, "/api/v1/auth/login",
		`{"email":"andrey@example.com","password":"muzyka2027"}`)
	unknownEmail := postJSON(t, handler, "/api/v1/auth/login",
		`{"email":"stanislav@example.com","password":"muzyka2026"}`)

	for _, w := range []*httptest.ResponseRecorder{wrongPassword, unknownEmail} {
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusUnauthorized, w.Body.String())
		}

		if got := decodeError(t, w); got.Error.Code != models.CodeInvalidCredentials {
			t.Errorf("code = %q, ожидался %q", got.Error.Code, models.CodeInvalidCredentials)
		}

		if len(w.Result().Cookies()) != 0 {
			t.Error("неудачный вход выставил cookie")
		}
	}

	if wrongPassword.Body.String() != unknownEmail.Body.String() {
		t.Errorf("ответы различаются:\nневерный пароль: %s\nнеизвестный email: %s",
			wrongPassword.Body.String(), unknownEmail.Body.String())
	}
}

func TestLoginErrors(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantErr    string
		wantFields []string
	}{
		{
			name:    "тело не разобралось",
			body:    `{"email": "andrey@example.com"`,
			wantErr: models.CodeInvalidJSON,
		},
		{
			name:       "нет полей",
			body:       `{}`,
			wantErr:    models.CodeValidationFailed,
			wantFields: []string{"email", "password"},
		},
		{
			name:       "пустой пароль",
			body:       `{"email":"andrey@example.com","password":""}`,
			wantErr:    models.CodeValidationFailed,
			wantFields: []string{"password"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := postJSON(t, newAuthHandler(), "/api/v1/auth/login", tt.body)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusBadRequest, w.Body.String())
			}

			got := decodeError(t, w)
			if got.Error.Code != tt.wantErr {
				t.Errorf("code = %q, ожидался %q", got.Error.Code, tt.wantErr)
			}
			for _, field := range tt.wantFields {
				if got.Error.Fields[field] == "" {
					t.Errorf("в fields нет поля %q, получено: %v", field, got.Error.Fields)
				}
			}
		})
	}
}

// На входе проверяется существующий пароль, а не требования к новым паролям.
func TestLoginDoesNotApplyPasswordRules(t *testing.T) {
	w := postJSON(t, newAuthHandler(), "/api/v1/auth/login",
		`{"email":"andrey@example.com","password":"123"}`)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusUnauthorized, w.Body.String())
	}
}

func TestLoginSessionFailure(t *testing.T) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)

	hasher := auth.NewBcryptHasherWithCost(auth.MinCost)
	hash, err := hasher.Hash("muzyka2026")
	if err != nil {
		t.Fatalf("хеширование пароля: %v", err)
	}

	users := &authUsersFixture{}
	if _, err := users.Create(context.Background(), models.User{
		Email:        "andrey@example.com",
		PasswordHash: hash,
		DisplayName:  "Андрей",
	}); err != nil {
		t.Fatalf("подготовка аккаунта: %v", err)
	}

	api := newRepositoryTestAPI(&testConfig, &repositoryTestDeps{Users: users, Sessions: failingSessionRepo{}, Hasher: hasher})
	handler := middleware.Chain(api.Routes(), middleware.WithJSONErrors)

	w := postJSON(t, handler, "/api/v1/auth/login",
		`{"email":"andrey@example.com","password":"muzyka2026"}`)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusInternalServerError, w.Body.String())
	}

	if got := decodeError(t, w); got.Error.Code != models.CodeInternal {
		t.Errorf("code = %q, ожидался %q", got.Error.Code, models.CodeInternal)
	}
}

func TestSignupCreatesUsableSession(t *testing.T) {
	sessions := &authSessionsFixture{}
	api := newRepositoryTestAPI(&testConfig, &repositoryTestDeps{
		Users:    &authUsersFixture{},
		Sessions: sessions,
		Hasher:   auth.NewBcryptHasherWithCost(auth.MinCost),
	})
	handler := middleware.Chain(api.Routes(), middleware.WithJSONErrors)

	w := postJSON(t, handler, "/api/v1/auth/signup",
		`{"email":"andrey@example.com","password":"muzyka2026","display_name":"Андрей"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var user userResponse
	if err := json.Unmarshal(w.Body.Bytes(), &user); err != nil {
		t.Fatalf("тело не разобралось: %v, тело: %s", err, w.Body.String())
	}

	session, err := sessions.GetByID(context.Background(), models.SessionID(sessionCookie(t, w).Value))
	if err != nil {
		t.Fatalf("сессия из cookie не нашлась в хранилище: %v", err)
	}

	if owner := string(session.UserID); owner != user.ID {
		t.Errorf("сессия принадлежит пользователю %s, а зарегистрирован %s", owner, user.ID)
	}

	if session.IsExpired(time.Now()) {
		t.Error("выданная сессия уже просрочена")
	}
}

func requestWithCookie(t *testing.T, handler http.Handler, method, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	return w
}

func TestSessionLifecycle(t *testing.T) {
	handler := newAuthHandler()

	signup := postJSON(t, handler, "/api/v1/auth/signup",
		`{"email":"andrey@example.com","password":"muzyka2026","display_name":"Андрей"}`)
	if signup.Code != http.StatusCreated {
		t.Fatalf("регистрация: статус = %d, тело: %s", signup.Code, signup.Body.String())
	}

	cookie := sessionCookie(t, signup)

	me := requestWithCookie(t, handler, http.MethodGet, "/api/v1/auth/me", cookie)
	if me.Code != http.StatusOK {
		t.Fatalf("me: статус = %d, ожидался %d, тело: %s", me.Code, http.StatusOK, me.Body.String())
	}

	var user userResponse
	if err := json.Unmarshal(me.Body.Bytes(), &user); err != nil {
		t.Fatalf("тело не разобралось: %v, тело: %s", err, me.Body.String())
	}

	if user.Email != "andrey@example.com" {
		t.Errorf("email = %q, ожидался %q", user.Email, "andrey@example.com")
	}

	logout := requestWithCookie(t, handler, http.MethodPost, "/api/v1/auth/logout", cookie)
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout: статус = %d, ожидался %d", logout.Code, http.StatusNoContent)
	}

	if cleared := sessionCookie(t, logout); cleared.MaxAge >= 0 {
		t.Errorf("MaxAge = %d, ожидалось отрицательное: cookie должна гаснуть", cleared.MaxAge)
	}

	after := requestWithCookie(t, handler, http.MethodGet, "/api/v1/auth/me", cookie)
	if after.Code != http.StatusUnauthorized {
		t.Errorf("me после выхода: статус = %d, ожидался %d", after.Code, http.StatusUnauthorized)
	}
}

func TestMeUnauthorizedCases(t *testing.T) {
	sessions := &authSessionsFixture{}
	api := newRepositoryTestAPI(&testConfig, &repositoryTestDeps{
		Users:    &authUsersFixture{},
		Sessions: sessions,
		Hasher:   auth.NewBcryptHasherWithCost(auth.MinCost),
	})
	handler := middleware.Chain(api.Routes(), middleware.WithJSONErrors)

	ctx := context.Background()
	expired := models.Session{ID: auth.NewSessionID(), UserID: models.UserID(uuid.New().String()), ExpiresAt: time.Now().Add(-time.Minute)}
	orphan := models.Session{ID: auth.NewSessionID(), UserID: models.UserID(uuid.New().String()), ExpiresAt: time.Now().Add(time.Hour)}

	for _, session := range []models.Session{expired, orphan} {
		if err := sessions.Create(ctx, session); err != nil {
			t.Fatalf("подготовка сессии: %v", err)
		}
	}

	tests := []struct {
		name   string
		cookie *http.Cookie
	}{
		{name: "cookie не прислана", cookie: nil},
		{name: "сессия неизвестна", cookie: &http.Cookie{Name: _sessionCookieName, Value: string(auth.NewSessionID())}},
		{name: "сессия истекла", cookie: &http.Cookie{Name: _sessionCookieName, Value: string(expired.ID)}},
		{name: "пользователь удалён", cookie: &http.Cookie{Name: _sessionCookieName, Value: string(orphan.ID)}},
	}

	var first string
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := requestWithCookie(t, handler, http.MethodGet, "/api/v1/auth/me", tt.cookie)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusUnauthorized, w.Body.String())
			}

			if got := decodeError(t, w); got.Error.Code != models.CodeUnauthorized {
				t.Errorf("code = %q, ожидался %q", got.Error.Code, models.CodeUnauthorized)
			}

			if first == "" {
				first = w.Body.String()
			} else if w.Body.String() != first {
				t.Errorf("ответ отличается от остальных:\n%s\n%s", first, w.Body.String())
			}
		})
	}
}

func TestLogoutWithoutSession(t *testing.T) {
	w := requestWithCookie(t, newAuthHandler(), http.MethodPost, "/api/v1/auth/logout", nil)

	if w.Code != http.StatusNoContent {
		t.Fatalf("статус = %d, ожидался %d", w.Code, http.StatusNoContent)
	}

	if w.Body.Len() != 0 {
		t.Errorf("204 пришёл с телом: %s", w.Body.String())
	}
}

type brokenSessionRepo struct {
	ports.SessionRepository
}

func (brokenSessionRepo) GetByID(ctx context.Context, id models.SessionID) (models.Session, error) {
	return models.Session{}, errStorageDown
}

func (brokenSessionRepo) Delete(ctx context.Context, id models.SessionID) error {
	return errStorageDown
}

// Ошибка БД должна давать 500, а не ответ об отсутствии авторизации.
func TestSessionStorageFailure(t *testing.T) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)

	handler := newHandlerWithSessions(brokenSessionRepo{})
	cookie := &http.Cookie{Name: _sessionCookieName, Value: string(auth.NewSessionID())}

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "me", method: http.MethodGet, path: "/api/v1/auth/me"},
		{name: "logout", method: http.MethodPost, path: "/api/v1/auth/logout"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := requestWithCookie(t, handler, tt.method, tt.path, cookie)

			if w.Code != http.StatusInternalServerError {
				t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusInternalServerError, w.Body.String())
			}

			if got := decodeError(t, w); got.Error.Code != models.CodeInternal {
				t.Errorf("code = %q, ожидался %q", got.Error.Code, models.CodeInternal)
			}

			if strings.Contains(w.Body.String(), "5432") {
				t.Errorf("детали ошибки хранилища ушли клиенту: %s", w.Body.String())
			}
		})
	}
}

func TestSessionCookieCrossOrigin(t *testing.T) {
	cfg := config.Config{
		SessionTTL:    time.Hour,
		AllowedOrigin: "https://music.example.com",
		CookieSecure:  true,
	}

	api := newRepositoryTestAPI(&cfg, &repositoryTestDeps{
		Users:    &authUsersFixture{},
		Sessions: &authSessionsFixture{},
		Hasher:   auth.NewBcryptHasherWithCost(auth.MinCost),
	})
	handler := middleware.Chain(api.Routes(), middleware.WithJSONErrors)

	w := postJSON(t, handler, "/api/v1/auth/signup",
		`{"email":"andrey@example.com","password":"muzyka2026","display_name":"Андрей"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("статус = %d, тело: %s", w.Code, w.Body.String())
	}

	cookie := sessionCookie(t, w)

	if cookie.SameSite != http.SameSiteNoneMode {
		t.Errorf("SameSite = %v, ожидался None", cookie.SameSite)
	}

	if !cookie.Secure {
		t.Error("cookie без Secure: с SameSite=None браузер её выбросит")
	}
}
