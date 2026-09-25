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

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/auth"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/config"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/storage"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/storage/memory"
)

// newSignupHandler собирает обработчики с настоящим хранилищем в памяти.
func newSignupHandler() http.Handler {
	api := New(config.Config{}, Deps{
		Users:  memory.NewUserStorage(),
		Hasher: auth.NewBcryptHasherWithCost(auth.MinCost),
	})
	return Wrap(api.Routes(), WithJSONErrors)
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
	handler := newSignupHandler()

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
	// Email приходит нормализованным.
	if got.Email != "andrey@example.com" {
		t.Errorf("email = %q, ожидался %q", got.Email, "andrey@example.com")
	}
	if got.DisplayName != "Андрей" {
		t.Errorf("display_name = %q, ожидался %q", got.DisplayName, "Андрей")
	}
	if got.AvatarURL != nil {
		t.Errorf("avatar_url = %v, ожидался null", *got.AvatarURL)
	}

	// Автоматического входа нет: cookie при регистрации не выставляется.
	if len(w.Result().Cookies()) != 0 {
		t.Error("при регистрации выставлена cookie, хотя автовхода быть не должно")
	}
}

// Самая важная проверка задачи: ни пароль, ни его хеш не должны уехать клиенту.
func TestSignupResponseHasNoPassword(t *testing.T) {
	handler := newSignupHandler()

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
		name     string
		body     string
		wantCode int
		wantErr  string
		// поля, которые обязаны быть в error.fields
		wantFields []string
	}{
		{
			name:     "тело не разобралось",
			body:     `{"email": "andrey@example.com"`,
			wantCode: http.StatusBadRequest,
			wantErr:  CodeInvalidJSON,
		},
		{
			name:     "пустое тело",
			body:     ``,
			wantCode: http.StatusBadRequest,
			wantErr:  CodeInvalidJSON,
		},
		{
			name:       "нет обязательных полей",
			body:       `{}`,
			wantCode:   http.StatusBadRequest,
			wantErr:    CodeValidationFailed,
			wantFields: []string{"email", "password", "display_name"},
		},
		{
			name:       "короткий пароль",
			body:       `{"email":"andrey@example.com","password":"123","display_name":"Андрей"}`,
			wantCode:   http.StatusBadRequest,
			wantErr:    CodeValidationFailed,
			wantFields: []string{"password"},
		},
		{
			name:       "некорректный email",
			body:       `{"email":"без-собаки","password":"muzyka2026","display_name":"Андрей"}`,
			wantCode:   http.StatusBadRequest,
			wantErr:    CodeValidationFailed,
			wantFields: []string{"email"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := postJSON(t, newSignupHandler(), "/api/v1/auth/signup", tt.body)

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
	handler := newSignupHandler()
	const body = `{"email":"andrey@example.com","password":"muzyka2026","display_name":"Андрей"}`

	if w := postJSON(t, handler, "/api/v1/auth/signup", body); w.Code != http.StatusCreated {
		t.Fatalf("первая регистрация: статус = %d, тело: %s", w.Code, w.Body.String())
	}

	w := postJSON(t, handler, "/api/v1/auth/signup", body)
	if w.Code != http.StatusConflict {
		t.Fatalf("вторая регистрация: статус = %d, ожидался %d", w.Code, http.StatusConflict)
	}

	got := decodeError(t, w)
	if got.Error.Code != CodeEmailTaken {
		t.Errorf("code = %q, ожидался %q", got.Error.Code, CodeEmailTaken)
	}
}

// Регистрация с тем же email в другом регистре — это тот же аккаунт.
func TestSignupEmailCaseInsensitive(t *testing.T) {
	handler := newSignupHandler()

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

// failingUserStorage изображает недоступное хранилище.
type failingUserStorage struct {
	storage.UserStorage
}

var errStorageDown = errors.New("хранилище недоступно: connection refused to 10.0.0.5:5432")

func (failingUserStorage) Create(ctx context.Context, user models.User) (models.User, error) {
	return models.User{}, errStorageDown
}

func TestSignupStorageFailure(t *testing.T) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)

	api := New(config.Config{}, Deps{
		Users:  failingUserStorage{},
		Hasher: auth.NewBcryptHasherWithCost(auth.MinCost),
	})
	handler := Wrap(api.Routes(), WithJSONErrors)

	w := postJSON(t, handler, "/api/v1/auth/signup",
		`{"email":"andrey@example.com","password":"muzyka2026","display_name":"Андрей"}`)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("статус = %d, ожидался %d", w.Code, http.StatusInternalServerError)
	}

	got := decodeError(t, w)
	if got.Error.Code != CodeInternal {
		t.Errorf("code = %q, ожидался %q", got.Error.Code, CodeInternal)
	}
	// Подробности сбоя остаются в логе: адрес базы клиенту знать незачем.
	if strings.Contains(w.Body.String(), "5432") {
		t.Errorf("детали ошибки хранилища ушли клиенту: %s", w.Body.String())
	}
}
