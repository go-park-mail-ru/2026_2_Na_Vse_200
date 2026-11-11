package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"2026_2_Na_Vse_200/internal/config"
	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"2026_2_Na_Vse_200/internal/models"
	authusecase "2026_2_Na_Vse_200/internal/usecase/auth"
)

type failedAuth struct {
	Usecase
	err error
}

func (s failedAuth) Signup(context.Context, authusecase.SignupInput) (authusecase.Result, error) {
	return authusecase.Result{}, s.err
}

func (s failedAuth) Login(context.Context, authusecase.LoginInput) (authusecase.Result, error) {
	return authusecase.Result{}, s.err
}

func (s failedAuth) Me(context.Context, models.SessionID) (models.User, error) {
	return models.User{}, s.err
}

func (s failedAuth) Logout(context.Context, models.SessionID) error {
	return s.err
}

// Ошибка обработчика и access-лог содержат один request_id.
func TestAuthErrorLogs(t *testing.T) {
	const requestID = "11111111-1111-4111-8111-111111111111"
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"signup", `{"password":"secret-password"}`, 500},
		{"login", `{"password":"secret-password"}`, 500},
		{"me", "", 500},
		{"logout", "", 500},
		{"signup", `{`, 400},
		{"login", `{`, 400},
	} {
		t.Run(tc.name+tc.body, func(t *testing.T) {
			var buf bytes.Buffer
			log := slog.New(slog.NewJSONHandler(&buf, nil))
			h := New(&config.Config{}, failedAuth{err: errors.New("db unavailable")}, log)
			handlers := map[string]http.HandlerFunc{"signup": h.Signup, "login": h.Login, "me": h.Me, "logout": h.Logout}
			r := httptest.NewRequest(http.MethodPost, "/auth/"+tc.name, strings.NewReader(tc.body))
			r.Header.Set("X-Request-ID", requestID)
			r.AddCookie(&http.Cookie{Name: _sessionCookieName, Value: "secret-session"})
			w := httptest.NewRecorder()
			middleware.Chain(handlers[tc.name], middleware.WithRequestID, middleware.WithLogging(log, "test")).ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("статус = %d, ожидался %d", w.Code, tc.status)
			}
			lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
			if len(lines) != 2 {
				t.Fatalf("ожидались лог ошибки и access-лог: %s", buf.String())
			}
			for i, line := range lines {
				var entry struct {
					RequestID string `json:"request_id"`
					Level     string `json:"level"`
					Error     string `json:"error"`
				}
				if err := json.Unmarshal([]byte(line), &entry); err != nil {
					t.Fatal(err)
				}
				if entry.RequestID != requestID {
					t.Errorf("request_id = %q", entry.RequestID)
				}
				if i == 0 && (entry.Error == "" || tc.status == 500 && entry.Level != "ERROR") {
					t.Errorf("неверный лог ошибки: %s", line)
				}
			}
			if strings.Contains(buf.String(), "secret-") {
				t.Fatalf("в лог попали пароль или cookie: %s", buf.String())
			}
		})
	}
}
