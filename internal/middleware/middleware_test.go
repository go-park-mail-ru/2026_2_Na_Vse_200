package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/apimessage"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/pkg/response"
)

// testLogger пишет лог в буфер.
func testLogger() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	return slog.New(slog.NewJSONHandler(&buf, nil)), &buf
}

// decodeLog разбирает последнюю строку лога как набор полей.
func decodeLog(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) == 0 || lines[0] == "" {
		t.Fatal("в лог ничего не записано")
	}

	var fields map[string]any
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &fields); err != nil {
		t.Fatalf("строка лога не разобралась как JSON: %v, строка: %s", err, lines[len(lines)-1])
	}
	return fields
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

func TestWithRequestID(t *testing.T) {
	var seen string
	handler := WithRequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = RequestIDFromContext(r.Context())
	}))

	t.Run("генерируется, если клиент ничего не прислал", func(t *testing.T) {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))

		if seen == "" {
			t.Fatal("идентификатор не попал в контекст запроса")
		}

		if got := w.Header().Get(HeaderRequestID); got != seen {
			t.Errorf("заголовок ответа = %q, в контексте %q — значения должны совпадать", got, seen)
		}
	})

	t.Run("свой идентификатор клиента переиспользуется", func(t *testing.T) {
		const client = "018f3b2a-4c5d-4e6f-8a9b-0c1d2e3f4a5b"

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set(HeaderRequestID, client)
		handler.ServeHTTP(httptest.NewRecorder(), req)

		if seen != client {
			t.Errorf("идентификатор = %q, ожидался присланный клиентом %q", seen, client)
		}
	})

	t.Run("мусор от клиента отбрасывается", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set(HeaderRequestID, "не-uuid-а-произвольный-текст")
		handler.ServeHTTP(httptest.NewRecorder(), req)

		if strings.Contains(seen, "произвольный") {
			t.Errorf("в лог уехало значение клиента без проверки: %q", seen)
		}

		if seen == "" {
			t.Error("взамен мусора не сгенерирован свой идентификатор")
		}
	})
}

func TestWithLogging(t *testing.T) {
	logger, buf := testLogger()

	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}),
		WithRequestID,
		WithLogging(logger, "monolith/middleware"),
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup?utm=test", nil)
	req.Header.Set("User-Agent", "curl/8.0")
	req.Header.Set("X-Real-IP", "203.0.113.7")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	fields := decodeLog(t, buf)

	for _, key := range []string{
		"request_id", "method", "url", "host", "remote_addr", "real_ip",
		"user_agent", "content_length", "status", "start_time",
		"duration_human_readable", "duration_ms", "handled_by",
	} {
		if _, ok := fields[key]; !ok {
			t.Errorf("в логе нет поля %q, записано: %v", key, fields)
		}
	}

	if fields["method"] != http.MethodPost {
		t.Errorf("method = %v, ожидался %v", fields["method"], http.MethodPost)
	}

	if fields["status"] != float64(http.StatusCreated) {
		t.Errorf("status = %v, ожидался %d", fields["status"], http.StatusCreated)
	}
	// В url должна попасть и строка запроса.
	if url, _ := fields["url"].(string); !strings.Contains(url, "utm=test") {
		t.Errorf("url = %v, ожидалась строка запроса целиком", fields["url"])
	}

	if fields["real_ip"] != "203.0.113.7" {
		t.Errorf("real_ip = %v, ожидался адрес из X-Real-IP", fields["real_ip"])
	}

	if fields["handled_by"] != "monolith/middleware" {
		t.Errorf("handled_by = %v, ожидался monolith/middleware", fields["handled_by"])
	}

	if id, _ := fields["request_id"].(string); id == "" {
		t.Error("request_id пуст — лог нельзя связать с конкретным запросом")
	}
}

func TestRealIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		want       string
	}{
		{
			name:       "без заголовков берём адрес соединения без порта",
			remoteAddr: "192.0.2.10:54321",
			want:       "192.0.2.10",
		},
		{
			name:       "X-Real-IP важнее адреса соединения",
			remoteAddr: "10.0.0.1:443",
			headers:    map[string]string{"X-Real-IP": "203.0.113.7"},
			want:       "203.0.113.7",
		},
		{
			name:       "из цепочки X-Forwarded-For берём первый адрес",
			remoteAddr: "10.0.0.1:443",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.7, 70.41.3.18, 10.0.0.1"},
			want:       "203.0.113.7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			if got := RealIP(req); got != tt.want {
				t.Errorf("RealIP = %q, ожидался %q", got, tt.want)
			}
		})
	}
}

// Паника не должна ронять процесс: клиент получает 500, подробности — в лог.
func TestWithRecover(t *testing.T) {
	logger, buf := testLogger()

	panicking := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("что-то пошло не так")
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Порядок как в main: recover снаружи всех остальных обёрток.
	Chain(panicking, WithRecover(logger), WithRequestID).ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("статус = %d, ожидался %d", w.Code, http.StatusInternalServerError)
	}

	if got := decodeError(t, w); got.Error.Code != apimessage.CodeInternal {
		t.Errorf("code = %q, ожидался %q", got.Error.Code, apimessage.CodeInternal)
	}

	if body := w.Body.String(); strings.Contains(body, "что-то пошло не так") {
		t.Errorf("текст паники ушёл клиенту: %s", body)
	}

	fields := decodeLog(t, buf)
	if fields["stack"] == nil {
		t.Error("в логе нет стека — по такой записи аварию не разобрать")
	}

	if id, _ := fields["request_id"].(string); id == "" {
		t.Error("в записи о панике нет request_id")
	}
}

func TestWithCORS(t *testing.T) {
	const allowed = "http://localhost:3000"

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := WithCORS(allowed)(okHandler)

	t.Run("разрешённый origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set("Origin", allowed)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		got := w.Header().Get("Access-Control-Allow-Origin")
		if got != allowed {
			t.Errorf("Allow-Origin = %q, ожидался %q", got, allowed)
		}

		if got == "*" {
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

// ServeMux отвечает текстом "404 page not found" — обёртка подменяет его на JSON.
func TestWithJSONErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := WithJSONErrors(mux)

	tests := []struct {
		name     string
		method   string
		path     string
		wantCode int
		wantErr  string
	}{
		{
			name:     "неизвестный путь",
			method:   http.MethodGet,
			path:     "/api/v1/nope",
			wantCode: http.StatusNotFound,
			wantErr:  apimessage.CodeNotFound,
		},
		{
			name:     "чужой метод",
			method:   http.MethodPost,
			path:     "/health",
			wantCode: http.StatusMethodNotAllowed,
			wantErr:  apimessage.CodeMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Fatalf("статус = %d, ожидался %d", w.Code, tt.wantCode)
			}

			if got := w.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
				t.Errorf("Content-Type = %q, ответ должен быть JSON, а не текстом маршрутизатора", got)
			}

			if got := decodeError(t, w); got.Error.Code != tt.wantErr {
				t.Errorf("code = %q, ожидался %q", got.Error.Code, tt.wantErr)
			}
		})
	}
}

// Успешный ответ обёртка не трогает.
func TestWithJSONErrorsPassesSuccess(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	WithJSONErrors(ok).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("статус = %d, ожидался %d", w.Code, http.StatusOK)
	}

	if body := w.Body.String(); !strings.Contains(body, "\"status\":\"ok\"") {
		t.Errorf("тело успешного ответа изменено: %s", body)
	}
}

// Первая обёртка в списке должна оказаться самой внешней.
func TestChainOrder(t *testing.T) {
	var order []string

	mark := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}

	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "handler")
		}),
		mark("первая"),
		mark("вторая"),
	)

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	want := []string{"первая", "вторая", "handler"}
	if len(order) != len(want) {
		t.Fatalf("порядок вызовов = %v, ожидался %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("порядок вызовов = %v, ожидался %v", order, want)
		}
	}
}
