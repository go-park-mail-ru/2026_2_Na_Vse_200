package handlers

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// Паника в обработчике не должна ронять процесс: клиент получает 500 в нашем формате.
func TestWithRecover(t *testing.T) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)

	panicking := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("что-то пошло не так")
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	WithRecover(panicking).ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("статус = %d, ожидался %d", w.Code, http.StatusInternalServerError)
	}

	got := decodeError(t, w)
	if got.Error.Code != CodeInternal {
		t.Errorf("code = %q, ожидался %q", got.Error.Code, CodeInternal)
	}
	// Наружу не должно утечь содержимое паники.
	if body := w.Body.String(); strings.Contains(body, "что-то пошло не так") {
		t.Errorf("текст паники ушёл клиенту: %s", body)
	}
}

func TestWithCORS(t *testing.T) {
	const allowed = "http://localhost:3000"

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := WithCORS([]string{allowed})(okHandler)

	t.Run("разрешённый origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set("Origin", allowed)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if got := w.Header().Get("Access-Control-Allow-Origin"); got != allowed {
			t.Errorf("Allow-Origin = %q, ожидался %q", got, allowed)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got == "*" {
			t.Error("Allow-Origin = *, это несовместимо с Allow-Credentials: cookie ходить не будут")
		}
		if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
			t.Errorf("Allow-Credentials = %q, ожидался %q", got, "true")
		}
	})

	t.Run("чужой origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set("Origin", "http://evil.example.com")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("Allow-Origin = %q, ожидалось, что чужому origin мы ничего не разрешаем", got)
		}
	})

	t.Run("preflight OPTIONS", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/health", nil)
		req.Header.Set("Origin", allowed)
		req.Header.Set("Access-Control-Request-Method", http.MethodGet)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("статус = %d, ожидался %d", w.Code, http.StatusNoContent)
		}
		if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
			t.Error("Allow-Methods пуст, браузер не пропустит основной запрос")
		}
	})
}
