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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/apimessage"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/auth"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/config"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository/memory"
)

// testConfig — настройки, при которых выданная сессия живёт дольше теста.
var testConfig = config.Config{SessionTTL: time.Hour}

// newAuthHandler собирает обработчики с настоящими хранилищами в памяти.
func newAuthHandler() http.Handler {
	api := New(&testConfig, &Deps{
		Users:    memory.NewUserRepo(),
		Sessions: memory.NewSessionRepo(),
		Hasher:   auth.NewBcryptHasherWithCost(auth.MinCost),
	})
	return middleware.Chain(api.Routes(), middleware.WithJSONErrors)
}

// sessionCookie возвращает cookie сессии из ответа.
func sessionCookie(t *testing.T, w *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()

	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == sessionCookieName {
			return cookie
		}
	}

	t.Fatalf("в ответе нет cookie %q", sessionCookieName)
	return nil
}

// postJSON отправляет запрос с телом и возвращает ответ.
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

// Ни пароль, ни его хеш не должны уехать клиенту.
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
			wantErr:  apimessage.CodeInvalidJSON,
		},
		{
			name:     "пустое тело",
			body:     ``,
			wantCode: http.StatusBadRequest,
			wantErr:  apimessage.CodeInvalidJSON,
		},
		{
			name:       "нет обязательных полей",
			body:       `{}`,
			wantCode:   http.StatusBadRequest,
			wantErr:    apimessage.CodeValidationFailed,
			wantFields: []string{"email", "password", "display_name"},
		},
		{
			name:       "короткий пароль",
			body:       `{"email":"andrey@example.com","password":"123","display_name":"Андрей"}`,
			wantCode:   http.StatusBadRequest,
			wantErr:    apimessage.CodeValidationFailed,
			wantFields: []string{"password"},
		},
		{
			name:       "некорректный email",
			body:       `{"email":"без-собаки","password":"muzyka2026","display_name":"Андрей"}`,
			wantCode:   http.StatusBadRequest,
			wantErr:    apimessage.CodeValidationFailed,
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
	if got.Error.Code != apimessage.CodeEmailTaken {
		t.Errorf("code = %q, ожидался %q", got.Error.Code, apimessage.CodeEmailTaken)
	}
}

// Регистрация с тем же email в другом регистре — это тот же аккаунт.
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

// failingUserRepo изображает недоступное хранилище.
type failingUserRepo struct {
	repository.UserRepositoryInterface
}

var errStorageDown = errors.New("хранилище недоступно: connection refused to 10.0.0.5:5432")

func (failingUserRepo) Create(ctx context.Context, user models.User) (models.User, error) {
	return models.User{}, errStorageDown
}

func TestSignupStorageFailure(t *testing.T) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)

	api := New(&testConfig, &Deps{
		Users:    failingUserRepo{},
		Sessions: memory.NewSessionRepo(),
		Hasher:   auth.NewBcryptHasherWithCost(auth.MinCost),
	})
	handler := middleware.Chain(api.Routes(), middleware.WithJSONErrors)

	w := postJSON(t, handler, "/api/v1/auth/signup",
		`{"email":"andrey@example.com","password":"muzyka2026","display_name":"Андрей"}`)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("статус = %d, ожидался %d", w.Code, http.StatusInternalServerError)
	}

	got := decodeError(t, w)
	if got.Error.Code != apimessage.CodeInternal {
		t.Errorf("code = %q, ожидался %q", got.Error.Code, apimessage.CodeInternal)
	}

	if strings.Contains(w.Body.String(), "5432") {
		t.Errorf("детали ошибки хранилища ушли клиенту: %s", w.Body.String())
	}
}

// failingSessionRepo изображает недоступное хранилище сессий.
type failingSessionRepo struct {
	repository.SessionRepositoryInterface
}

func (failingSessionRepo) Create(ctx context.Context, session models.Session) error {
	return errStorageDown
}

// newHandlerWithSessions собирает обработчики с подменённым хранилищем сессий.
func newHandlerWithSessions(sessions repository.SessionRepositoryInterface) http.Handler {
	api := New(&testConfig, &Deps{
		Users:    memory.NewUserRepo(),
		Sessions: sessions,
		Hasher:   auth.NewBcryptHasherWithCost(auth.MinCost),
	})
	return middleware.Chain(api.Routes(), middleware.WithJSONErrors)
}

// Аккаунт создан, значит регистрация удалась — даже если автовход не вышел.
func TestSignupSessionFailure(t *testing.T) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)

	w := postJSON(t, newHandlerWithSessions(failingSessionRepo{}), "/api/v1/auth/signup",
		`{"email":"andrey@example.com","password":"muzyka2026","display_name":"Андрей"}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	if len(w.Result().Cookies()) != 0 {
		t.Error("выставлена cookie, хотя сессия не сохранилась")
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

// Неизвестный email и неверный пароль обязаны отвечать одинаково: иначе
// по разнице ответов перебирают зарегистрированные адреса.
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

		if got := decodeError(t, w); got.Error.Code != apimessage.CodeInvalidCredentials {
			t.Errorf("code = %q, ожидался %q", got.Error.Code, apimessage.CodeInvalidCredentials)
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
			wantErr: apimessage.CodeInvalidJSON,
		},
		{
			name:       "нет полей",
			body:       `{}`,
			wantErr:    apimessage.CodeValidationFailed,
			wantFields: []string{"email", "password"},
		},
		{
			name:       "пустой пароль",
			body:       `{"email":"andrey@example.com","password":""}`,
			wantErr:    apimessage.CodeValidationFailed,
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

// Короткий пароль на входе — это не ошибка валидации, а неверные данные:
// правила могли ужесточить уже после регистрации аккаунта.
func TestLoginDoesNotApplyPasswordRules(t *testing.T) {
	w := postJSON(t, newAuthHandler(), "/api/v1/auth/login",
		`{"email":"andrey@example.com","password":"123"}`)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusUnauthorized, w.Body.String())
	}
}

// В отличие от регистрации, вход без сессии смысла не имеет.
func TestLoginSessionFailure(t *testing.T) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)

	handler := newHandlerWithSessions(failingSessionRepo{})

	if w := postJSON(t, handler, "/api/v1/auth/signup",
		`{"email":"andrey@example.com","password":"muzyka2026","display_name":"Андрей"}`); w.Code != http.StatusCreated {
		t.Fatalf("регистрация: статус = %d, тело: %s", w.Code, w.Body.String())
	}

	w := postJSON(t, handler, "/api/v1/auth/login",
		`{"email":"andrey@example.com","password":"muzyka2026"}`)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusInternalServerError, w.Body.String())
	}

	if got := decodeError(t, w); got.Error.Code != apimessage.CodeInternal {
		t.Errorf("code = %q, ожидался %q", got.Error.Code, apimessage.CodeInternal)
	}
}

// Выданная при регистрации cookie должна указывать на настоящую сессию
// этого же пользователя, иначе автовход есть только на бумаге.
func TestSignupCreatesUsableSession(t *testing.T) {
	sessions := memory.NewSessionRepo()
	api := New(&testConfig, &Deps{
		Users:    memory.NewUserRepo(),
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

	session, err := sessions.GetByID(context.Background(), sessionCookie(t, w).Value)
	if err != nil {
		t.Fatalf("сессия из cookie не нашлась в хранилище: %v", err)
	}

	if owner := strconv.FormatInt(int64(session.UserID), 10); owner != user.ID {
		t.Errorf("сессия принадлежит пользователю %s, а зарегистрирован %s", owner, user.ID)
	}

	if session.IsExpired(time.Now()) {
		t.Error("выданная сессия уже просрочена")
	}
}
