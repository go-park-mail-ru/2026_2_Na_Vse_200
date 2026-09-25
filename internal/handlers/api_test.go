package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/config"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/pkg/apimessage"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/pkg/response"
)

// newTestHandler собирает маршруты с подменой текстовых ошибок на JSON.
func newTestHandler() http.Handler {
	api := New(config.Config{}, Deps{})
	return middleware.Chain(api.Routes(), middleware.WithJSONErrors)
}

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	newTestHandler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /health: статус = %d, ожидался %d", w.Code, http.StatusOK)
	}

	var body healthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("GET /health: тело не разобралось: %v, тело: %s", err, w.Body.String())
	}
	if body.Status != "ok" {
		t.Errorf("GET /health: status = %q, ожидался %q", body.Status, "ok")
	}
}

func TestRoutesErrors(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		wantCode int
		wantErr  string
	}{
		{
			name:     "неизвестный путь под api",
			method:   http.MethodGet,
			path:     "/api/v1/nope",
			wantCode: http.StatusNotFound,
			wantErr:  apimessage.CodeNotFound,
		},
		{
			name:     "корень сайта больше не обрабатывается",
			method:   http.MethodGet,
			path:     "/",
			wantCode: http.StatusNotFound,
			wantErr:  apimessage.CodeNotFound,
		},
		{
			name:     "чужой метод на существующем адресе",
			method:   http.MethodPost,
			path:     "/health",
			wantCode: http.StatusMethodNotAllowed,
			wantErr:  apimessage.CodeMethodNotAllowed,
		},
	}

	handler := newTestHandler()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("%s %s: статус = %d, ожидался %d", tt.method, tt.path, w.Code, tt.wantCode)
			}

			// Ответ должен быть JSON, а не текстом ServeMux.
			gotType := w.Header().Get("Content-Type")
			wantType := "application/json; charset=utf-8"
			if gotType != wantType {
				t.Errorf("%s %s: Content-Type = %q, ожидался %q", tt.method, tt.path, gotType, wantType)
			}

			got := decodeError(t, w)
			if got.Error.Code != tt.wantErr {
				t.Errorf("%s %s: code = %q, ожидался %q", tt.method, tt.path, got.Error.Code, tt.wantErr)
			}
			if got.Error.Message == "" {
				t.Errorf("%s %s: message пустой", tt.method, tt.path)
			}
		})
	}
}

// ServeMux ставит Allow до подмены тела — проверяем, что он не потерялся.
func TestMethodNotAllowedKeepsAllowHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()

	newTestHandler().ServeHTTP(w, req)

	if allow := w.Header().Get("Allow"); allow == "" {
		t.Error("POST /health: заголовок Allow пуст, клиент не узнает разрешённые методы")
	}
}

// decodeError разбирает тело ответа как ошибку единого формата API.
func decodeError(t *testing.T, w *httptest.ResponseRecorder) response.ErrorResponse {
	t.Helper()

	var got response.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("тело не разобралось как ошибка API: %v, тело: %s", err, w.Body.String())
	}
	return got
}
