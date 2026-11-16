package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/security"
)

func TestWithCSRF(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := WithCSRF(ok)

	t.Run("get без токена", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/home", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d", w.Code)
		}
	})

	t.Run("post без токена", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d", w.Code)
		}
		got := decodeError(t, w)
		if got.Error.Code != models.CodeCSRFRejected {
			t.Fatalf("code = %q", got.Error.Code)
		}
	})

	t.Run("post с совпадающими токенами", func(t *testing.T) {
		token := "abc123"
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		req.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: token})
		req.Header.Set(security.CSRFHeaderName, token)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("post с разными токенами", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/v1/users/1", nil)
		req.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "one"})
		req.Header.Set(security.CSRFHeaderName, "two")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d", w.Code)
		}
	})
}
