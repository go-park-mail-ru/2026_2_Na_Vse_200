package onlineconfadmin

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/onlineconf"
	"2026_2_Na_Vse_200/internal/security"
)

func TestProtectedAdminSPAAndProxy(t *testing.T) {
	const adminID = "11111111-1111-4111-8111-111111111111"
	const payload = `{"kind":"config","value":3}`
	const saved = `{"kind":"config","value":3,"scope":"catalog","key":"popular_tracks_limit"}`
	var upstreamCalls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls.Add(1)
		for _, name := range []string{"Cookie", security.CSRFHeaderName, "Authorization"} {
			if value := r.Header.Get(name); value != "" {
				t.Errorf("forwarded client header %s", name)
			}
		}
		switch r.Method {
		case http.MethodGet:
			if r.URL.Path != "/internal/v1/entries" || r.URL.Query().Get("q") != "popular tracks" {
				t.Errorf("unexpected list URL: %s", r.URL)
			}
			_, _ = io.WriteString(w, `{"entries":[`+saved+`]}`)
		case http.MethodPut:
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != payload || r.URL.Path != "/internal/v1/entries/catalog/popular_tracks_limit" ||
				r.Header.Get("X-OnlineConf-Actor") != adminID {
				t.Errorf("unexpected save request URL=%s body=%s actor=%q err=%v", r.URL, body, r.Header.Get("X-OnlineConf-Actor"), err)
			}
			_, _ = io.WriteString(w, saved)
		}
	}))
	defer upstream.Close()
	client, err := onlineconf.NewClient(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	dist := t.TempDir()
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte("admin SPA"), 0600); err != nil {
		t.Fatal(err)
	}
	handler, err := New(client, dist, adminID)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	handler.Register(mux, middleware.WithAuth("session_id", func(_ context.Context, id models.SessionID) (models.Session, error) {
		switch id {
		case "admin":
			return models.Session{UserID: adminID}, nil
		case "user":
			return models.Session{UserID: "22222222-2222-4222-8222-222222222222"}, nil
		default:
			return models.Session{}, models.ErrSessionNotFound
		}
	}))

	for _, tc := range []struct {
		name, method, path, session string
		csrf                        bool
		status                      int
		body                        string
	}{
		{"guest", http.MethodGet, "/admin/", "", false, 401, `"unauthorized"`},
		{"user SPA", http.MethodGet, "/admin/settings", "user", false, 403, `"forbidden"`},
		{"user API", http.MethodGet, APIPrefix + "entries", "user", false, 403, `"forbidden"`},
		{"admin SPA", http.MethodGet, "/admin/settings", "admin", false, 200, "admin SPA"},
		{"missing asset", http.MethodGet, "/admin/assets/missing.js", "admin", false, 404, ""},
		{"no CSRF", http.MethodPut, APIPrefix + "entries/catalog/popular_tracks_limit", "admin", false, 403, `"csrf_rejected"`},
		{"admin list", http.MethodGet, APIPrefix + "entries?q=popular+tracks", "admin", false, 200, `"entries"`},
		{"admin save", http.MethodPut, APIPrefix + "entries/catalog/popular_tracks_limit", "admin", true, 200, saved},
		{"encoded slash", http.MethodPut, APIPrefix + "entries/catalog%2Fextra/popular_tracks_limit", "admin", true, 400, `"validation_failed"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(payload))
			if tc.session != "" {
				req.AddCookie(&http.Cookie{Name: "session_id", Value: tc.session})
			}
			req.Header.Set("X-OnlineConf-Actor", "browser supplied ID")
			req.Header.Set("Authorization", "Bearer browser-token")
			if tc.csrf {
				req.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "token"})
				req.Header.Set(security.CSRFHeaderName, "token")
			}
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != tc.status || !strings.Contains(rec.Body.String(), tc.body) {
				t.Fatalf("got %d %s; want %d containing %q", rec.Code, rec.Body, tc.status, tc.body)
			}
			if got := rec.Header().Get("Cache-Control"); got != "private, no-store" {
				t.Fatalf("Cache-Control: %q", got)
			}
		})
	}
	if got := upstreamCalls.Load(); got != 2 {
		t.Errorf("upstream calls = %d, want 2", got)
	}
}
