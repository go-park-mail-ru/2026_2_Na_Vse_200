package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestHTTPMetrics(t *testing.T) {
	metrics := NewHTTPMetrics("test-api")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok")
	})
	mux.HandleFunc("POST /auth/login", func(http.ResponseWriter, *http.Request) {
		t.Error("обработчик входа не должен вызываться при отказе CSRF")
	})
	mux.HandleFunc("GET /artists/{id}", http.NotFound)
	mux.HandleFunc("POST /artists/import", func(http.ResponseWriter, *http.Request) {
		t.Error("обработчик импорта не должен вызываться при preflight")
	})
	mux.HandleFunc("GET /panic", func(http.ResponseWriter, *http.Request) {
		panic("test panic")
	})
	mux.HandleFunc("GET /audio", func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "sample.mp3", time.Time{}, strings.NewReader("abcdef"))
	})
	mux.HandleFunc("GET /health", func(http.ResponseWriter, *http.Request) {})
	const origin = "http://127.0.0.1:5173"
	handler := metrics.Wrap(mux, WithRequestID, WithLogging(zerolog.Nop(), "test"), WithRecover(),
		WithCORS(origin, "GET, POST, OPTIONS", "Content-Type"), WithCSRF, WithJSONErrors)

	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{http.MethodGet, "/users/123?secret=value", 200},
		{http.MethodGet, "/users/456", 200},
		{http.MethodGet, "/panic", 500},
		{http.MethodPost, "/auth/login", 403},
		{http.MethodGet, "/auth/login", 405},
		{http.MethodGet, "/artists/987", 404},
		{http.MethodGet, "/missing", 404},
		{http.MethodGet, "/health", 200},
		{http.MethodGet, "/metrics", 404},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != tc.status {
			t.Errorf("%s %s: статус %d, ожидался %d", tc.method, tc.path, w.Code, tc.status)
		}
	}
	preflight := httptest.NewRequest(http.MethodOptions, "/artists/import", nil)
	preflight.Header.Set("Origin", origin)
	preflight.Header.Set("Access-Control-Request-Method", http.MethodPost)
	preflightResponse := httptest.NewRecorder()
	handler.ServeHTTP(preflightResponse, preflight)
	if preflightResponse.Code != http.StatusNoContent {
		t.Errorf("preflight: статус %d, ожидался %d", preflightResponse.Code, http.StatusNoContent)
	}
	req := httptest.NewRequest(http.MethodGet, "/audio", nil)
	req.Header.Set("Range", "bytes=1-3")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusPartialContent || w.Body.String() != "bcd" || w.Header().Get("Content-Range") != "bytes 1-3/6" {
		t.Fatalf("Range: статус %d, тело %q, заголовки %v", w.Code, w.Body, w.Header())
	}

	scrape := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(scrape, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := scrape.Body.String()
	for _, line := range []string{
		`http_requests_total{method="GET",route="/users/{id}",service="test-api",status="200"} 2`,
		`http_requests_total{method="GET",route="/panic",service="test-api",status="500"} 1`,
		`http_requests_total{method="POST",route="/auth/login",service="test-api",status="403"} 1`,
		`http_requests_total{method="GET",route="/auth/login",service="test-api",status="405"} 1`,
		`http_requests_total{method="OPTIONS",route="/artists/import",service="test-api",status="204"} 1`,
		`http_requests_total{method="GET",route="/artists/{id}",service="test-api",status="404"} 1`,
		`http_requests_total{method="GET",route="unmatched",service="test-api",status="404"} 1`,
		`http_requests_total{method="GET",route="/audio",service="test-api",status="206"} 1`,
		`http_requests_total{method="GET",route="/health",service="test-api",status="200"} 1`,
		`http_request_duration_seconds_count{method="GET",route="/users/{id}",service="test-api"} 2`,
		`http_request_duration_seconds_count{method="POST",route="/auth/login",service="test-api"} 1`,
		`http_request_duration_seconds_count{method="GET",route="/auth/login",service="test-api"} 1`,
		`http_request_duration_seconds_count{method="OPTIONS",route="/artists/import",service="test-api"} 1`,
	} {
		if !strings.Contains(body, line+"\n") {
			t.Errorf("в метриках нет строки %s", line)
		}
	}
	for _, value := range []string{"/users/123", "/users/456", "/artists/987", "/missing", "secret", "/metrics"} {
		if strings.Contains(body, value) {
			t.Errorf("в метрики попало значение %q", value)
		}
	}
}
