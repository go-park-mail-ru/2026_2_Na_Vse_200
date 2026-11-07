package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"2026_2_Na_Vse_200/internal/auth"
	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"2026_2_Na_Vse_200/internal/models"
)

type profileResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url"`
	CreatedAt   time.Time `json:"created_at"`
}

type countingUsers struct {
	inner        authUsersFixture
	getByIDCalls int
}

func (r *countingUsers) Create(ctx context.Context, user models.User) (models.User, error) {
	return r.inner.Create(ctx, user)
}

func (r *countingUsers) GetByID(ctx context.Context, id models.UserID) (models.User, error) {
	r.getByIDCalls++
	return r.inner.GetByID(ctx, id)
}

func (r *countingUsers) GetByEmail(ctx context.Context, email string) (models.User, error) {
	return r.inner.GetByEmail(ctx, email)
}

func (r *countingUsers) UpdateProfile(ctx context.Context, id models.UserID, email, displayName string) (models.User, error) {
	return r.inner.UpdateProfile(ctx, id, email, displayName)
}

func (r *countingUsers) SetAvatar(ctx context.Context, id models.UserID, fileID models.MediaFileID) (models.User, error) {
	return r.inner.SetAvatar(ctx, id, fileID)
}

func newProfileFixture(t *testing.T) (http.Handler, *http.Cookie, userResponse, *countingUsers) {
	t.Helper()

	users := &countingUsers{}
	api := newRepositoryTestAPI(&testConfig, &repositoryTestDeps{
		Users:    users,
		Sessions: &authSessionsFixture{},
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
	return handler, sessionCookie(t, w), user, users
}

func TestProfileOwner(t *testing.T) {
	handler, cookie, user, _ := newProfileFixture(t)

	tests := []struct {
		name string
		id   string
	}{
		{name: "id как выдан", id: user.ID},
		{name: "id в верхнем регистре", id: strings.ToUpper(user.ID)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := requestWithCookie(t, handler, http.MethodGet, "/api/v1/users/"+tt.id, cookie)
			if w.Code != http.StatusOK {
				t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusOK, w.Body.String())
			}

			var got profileResponse
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatalf("тело не разобралось: %v, тело: %s", err, w.Body.String())
			}
			if got.ID != user.ID {
				t.Errorf("id = %q, ожидался %q", got.ID, user.ID)
			}
			if got.Email != "andrey@example.com" {
				t.Errorf("email = %q", got.Email)
			}
			if got.DisplayName != "Андрей" {
				t.Errorf("display_name = %q", got.DisplayName)
			}
			if got.AvatarURL != nil {
				t.Errorf("avatar_url = %v, ожидался null", *got.AvatarURL)
			}
			if got.CreatedAt.IsZero() {
				t.Error("created_at пустой")
			}
		})
	}
}

func TestProfileForbiddenDoesNotReadUser(t *testing.T) {
	handler, cookie, _, users := newProfileFixture(t)

	w := requestWithCookie(t, handler, http.MethodGet,
		"/api/v1/users/33333333-3333-3333-3333-333333333333", cookie)
	if w.Code != http.StatusForbidden {
		t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
	if got := decodeError(t, w); got.Error.Code != models.CodeForbidden {
		t.Errorf("code = %q, ожидался %q", got.Error.Code, models.CodeForbidden)
	}
	if users.getByIDCalls != 0 {
		t.Fatalf("репозиторий вызван %d раз, ожидалось 0", users.getByIDCalls)
	}
}

func TestProfileUnauthorized(t *testing.T) {
	handler, _, user, _ := newProfileFixture(t)

	w := requestWithCookie(t, handler, http.MethodGet, "/api/v1/users/"+user.ID, nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusUnauthorized, w.Body.String())
	}
	if got := decodeError(t, w); got.Error.Code != models.CodeUnauthorized {
		t.Errorf("code = %q, ожидался %q", got.Error.Code, models.CodeUnauthorized)
	}
}

func TestProfileInvalidID(t *testing.T) {
	handler, cookie, _, _ := newProfileFixture(t)

	w := requestWithCookie(t, handler, http.MethodGet, "/api/v1/users/not-a-uuid", cookie)
	if w.Code != http.StatusNotFound {
		t.Fatalf("статус = %d, ожидался %d, тело: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
	if got := decodeError(t, w); got.Error.Code != models.CodeNotFound {
		t.Errorf("code = %q, ожидался %q", got.Error.Code, models.CodeNotFound)
	}
}

func TestProfileUpdate(t *testing.T) {
	handler, cookie, user, _ := newProfileFixture(t)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+user.ID, strings.NewReader(
		`{"email":"andrey@example.com","display_name":"Новое имя"}`,
	))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("статус = %d, тело: %s", w.Code, w.Body.String())
	}
	var got profileResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("json: %v", err)
	}
	if got.Email != "andrey@example.com" || got.DisplayName != "Новое имя" {
		t.Fatalf("got=%+v", got)
	}
}

func TestProfileUploadAvatar(t *testing.T) {
	handler, cookie, user, _ := newProfileFixture(t)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("avatar", "avatar.png")
	if err != nil {
		t.Fatal(err)
	}
	png := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	png = append(png, bytes.Repeat([]byte{1}, 64)...)
	if _, err := part.Write(png); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+user.ID+"/avatar", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("статус = %d, тело: %s", w.Code, w.Body.String())
	}
	var got profileResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("json: %v", err)
	}
	if got.AvatarURL == nil || !strings.HasPrefix(*got.AvatarURL, "/api/v1/media/") {
		t.Fatalf("avatar_url=%v", got.AvatarURL)
	}
}
