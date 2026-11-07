package profile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"2026_2_Na_Vse_200/internal/config"
	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"2026_2_Na_Vse_200/internal/models"
)

const ownerID = "77777777-0000-4000-8000-000000000001"

type accountUsecaseStub struct {
	calls int
	err   error
}

func (s *accountUsecaseStub) ChangeEmail(_ context.Context, _, id models.UserID, _, email string) (models.User, error) {
	s.calls++
	return models.User{ID: id, Email: email, DisplayName: "Name"}, s.err
}
func (s *accountUsecaseStub) ChangePassword(context.Context, models.UserID, models.UserID, string, string) error {
	s.calls++
	return s.err
}
func (s *accountUsecaseStub) Delete(context.Context, models.UserID, models.UserID, string) error {
	s.calls++
	return s.err
}

func TestAccountHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, body string
		auth                     bool
		err                      error
		status, calls            int
	}{
		{"email", "PUT", "/email", `{"current_password":"Old12345","email":"new@example.test"}`, true, nil, 200, 1},
		{"password", "PUT", "/password", `{"current_password":"Old12345","new_password":"New12345"}`, true, nil, 204, 1},
		{"delete", "DELETE", "", `{"current_password":"Old12345"}`, true, nil, 204, 1},
		{"guest", "DELETE", "", `{}`, false, nil, 401, 0},
		{"wrong password", "PUT", "/password", `{}`, true, models.ErrCurrentPassword, 400, 1},
		{"duplicate email", "PUT", "/email", `{}`, true, models.ErrEmailTaken, 409, 1},
		{"bad JSON", "DELETE", "", `{`, true, nil, 400, 0},
		{"trailing JSON", "DELETE", "", `{} {}`, true, nil, 400, 0},
		{"unknown field", "DELETE", "", `{"password":"wrong-name"}`, true, nil, 400, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &accountUsecaseStub{err: tc.err}
			h := NewAccount(stub, &config.Config{CookieSecure: true})
			mux := http.NewServeMux()
			mux.HandleFunc("PUT /users/{id}/email", h.ChangeEmail)
			mux.HandleFunc("PUT /users/{id}/password", h.ChangePassword)
			mux.HandleFunc("DELETE /users/{id}", h.Delete)
			protected := middleware.WithAuth("session_id", func(context.Context, models.SessionID) (models.Session, error) {
				return models.Session{UserID: ownerID, ExpiresAt: time.Now().Add(time.Hour)}, nil
			})(mux)
			r := httptest.NewRequest(tc.method, "/users/"+ownerID+tc.path, strings.NewReader(tc.body))
			if tc.auth {
				r.AddCookie(&http.Cookie{Name: "session_id", Value: "test"})
			}
			w := httptest.NewRecorder()
			protected.ServeHTTP(w, r)
			if w.Code != tc.status || stub.calls != tc.calls {
				t.Fatalf("status=%d calls=%d body=%s", w.Code, stub.calls, w.Body.String())
			}
			if tc.status == 204 {
				cookies := w.Result().Cookies()
				if len(cookies) != 1 || cookies[0].MaxAge != -1 || !cookies[0].HttpOnly || !cookies[0].Secure {
					t.Fatal("session cookie not cleared")
				}
			}
		})
	}
}
