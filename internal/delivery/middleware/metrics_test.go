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
	mux.HandleFunc("GET /panic", func(http.ResponseWriter, *http.Request) {
		panic("test panic")
	})
	mux.HandleFunc("GET /audio", func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "sample.mp3", time.Time{}, strings.NewReader("abcdef"))
	})
	mux.HandleFunc("GET /health", func(http.ResponseWriter, *http.Request) {})
	handler := metrics.Wrap(mux, WithRequestID, WithLogging(zerolog.Nop(), "test"), WithRecover(), WithCSRF, WithJSONErrors)

	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{http.MethodGet, "/users/123?secret=value", 200},
		{http.MethodGet, "/users/456", 200},
		{http.MethodGet, "/panic", 500},
		{http.MethodPost, "/users/123", 403},
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
		`http_requests_total{method="POST",route="unmatched",service="test-api",status="403"} 1`,
		`http_requests_total{method="GET",route="unmatched",service="test-api",status="404"} 1`,
		`http_requests_total{method="GET",route="/audio",service="test-api",status="206"} 1`,
		`http_requests_total{method="GET",route="/health",service="test-api",status="200"} 1`,
		`http_request_duration_seconds_count{method="GET",route="/users/{id}",service="test-api"} 2`,
	} {
		if !strings.Contains(body, line+"\n") {
			t.Errorf("в метриках нет строки %s", line)
		}
	}
	for _, value := range []string{"/users/123", "/users/456", "secret", "/metrics"} {
		if strings.Contains(body, value) {
			t.Errorf("в метрики попало значение %q", value)
		}
	}
}
