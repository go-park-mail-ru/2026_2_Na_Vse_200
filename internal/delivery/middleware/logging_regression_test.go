package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestRequestLoggerKeepsRequestIDAndHidesQuery(t *testing.T) {
	logger, buf := testLogger()
	handler := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zerolog.Ctx(r.Context()).Info().Str("operation", "test.handler").Msg("handler event")
		w.WriteHeader(http.StatusNoContent)
	}), WithRequestID, WithLogging(logger, "test"), WithRecover())
	ids := []string{"018f3b2a-4c5d-4e6f-8a9b-0c1d2e3f4a5b", "018f3b2a-4c5d-4e6f-8a9b-0c1d2e3f4a5c"}
	for _, id := range ids {
		buf.Reset()
		req := httptest.NewRequest(http.MethodGet, "/test?token=secret-query", nil)
		req.Header.Set(HeaderRequestID, id)
		req.Header.Set("Authorization", "Bearer secret-header")
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "secret-cookie"})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent || w.Header().Get(HeaderRequestID) != id {
			t.Fatalf("response: code=%d request_id=%q", w.Code, w.Header().Get(HeaderRequestID))
		}
		events := readRegressionEvents(t, buf)
		if len(events) != 2 {
			t.Fatalf("got %d events, want handler and access", len(events))
		}
		for _, event := range events {
			if event["request_id"] != id {
				t.Fatalf("request_id=%v, want %s", event["request_id"], id)
			}
		}
		if events[1]["status"] != float64(http.StatusNoContent) {
			t.Fatalf("access status=%v", events[1]["status"])
		}
		for _, secret := range []string{"secret-query", "secret-header", "secret-cookie"} {
			if strings.Contains(buf.String(), secret) {
				t.Errorf("log contains %s", secret)
			}
		}
	}
}

func TestPanicHasRecoveryAndAccessEvents(t *testing.T) {
	logger, buf := testLogger()
	handler := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("private panic detail")
	}), WithRequestID, WithLogging(logger, "test"), WithRecover())
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "private panic detail") {
		t.Fatalf("unsafe response: status=%d body=%s", w.Code, w.Body.String())
	}
	events := readRegressionEvents(t, buf)
	if len(events) != 2 {
		t.Fatalf("got %d events, want recovery and access", len(events))
	}
	if events[0]["level"] != "error" || events[0]["stack"] == nil {
		t.Fatalf("missing recovery detail: %v", events[0])
	}
	if events[1]["status"] != float64(http.StatusInternalServerError) || events[1]["level"] != "error" {
		t.Fatalf("access event=%v, ожидался статус 500 и уровень error", events[1])
	}
	for _, event := range events {
		if event["request_id"] != w.Header().Get(HeaderRequestID) {
			t.Fatalf("request_id mismatch: %v", event)
		}
	}
}

func TestStatusWriterKeepsFirstFinalStatus(t *testing.T) {
	w := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: w}
	sw.WriteHeader(http.StatusCreated)
	sw.WriteHeader(http.StatusInternalServerError)
	if sw.status != http.StatusCreated || w.Code != http.StatusCreated {
		t.Fatalf("recorded=%d actual=%d", sw.status, w.Code)
	}
}

func TestStatusWriterCountsPartialWrite(t *testing.T) {
	recorder := httptest.NewRecorder()
	writer := &statusWriter{ResponseWriter: partialResponseWriter{recorder}}
	n, err := writer.Write([]byte("hello"))
	if n != 3 || err != io.ErrShortWrite || writer.bytesWritten != 3 {
		t.Fatalf("Write вернул %d, %v, счётчик = %d", n, err, writer.bytesWritten)
	}
	if writer.status != http.StatusOK || recorder.Body.String() != "hel" {
		t.Fatalf("статус = %d, тело = %q", writer.status, recorder.Body.String())
	}
}

type partialResponseWriter struct{ http.ResponseWriter }

func (w partialResponseWriter) Write(body []byte) (int, error) {
	n, err := w.ResponseWriter.Write(body[:min(len(body), 3)])
	if err == nil && n < len(body) {
		err = io.ErrShortWrite
	}
	return n, err
}

func TestAccessLogRecordsLevelAndResponseBytes(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status int
		level  string
		parts  []string
		bytes  int
	}{
		{"ok", http.StatusOK, "info", []string{"ab", "cde"}, 5},
		{"no_content", http.StatusNoContent, "info", nil, 0},
		{"unauthorized", http.StatusUnauthorized, "warn", []string{"no"}, 2},
		{"server_error", http.StatusInternalServerError, "error", []string{"error"}, 5},
	} {
		t.Run(tt.name, func(t *testing.T) {
			logger, buf := testLogger()
			handler := WithLogging(logger, "test")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				for _, part := range tt.parts {
					if _, err := io.WriteString(w, part); err != nil {
						t.Fatal(err)
					}
				}
			}))
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("input"))
			handler.ServeHTTP(recorder, req)
			events := readRegressionEvents(t, buf)
			if len(events) != 1 || events[0]["level"] != tt.level || events[0]["status"] != float64(tt.status) ||
				events[0]["response_bytes"] != float64(tt.bytes) || events[0]["content_length"] != float64(5) {
				t.Fatalf("неверный access-log: %v", events)
			}
			if recorder.Code != tt.status || recorder.Body.String() != strings.Join(tt.parts, "") {
				t.Fatalf("ответ изменился: статус = %d, тело = %q", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestAccessLogLevelBoundaries(t *testing.T) {
	for _, tt := range []struct {
		status int
		want   zerolog.Level
	}{
		{0, zerolog.InfoLevel},
		{99, zerolog.InfoLevel},
		{399, zerolog.InfoLevel},
		{400, zerolog.WarnLevel},
		{499, zerolog.WarnLevel},
		{500, zerolog.ErrorLevel},
		{599, zerolog.ErrorLevel},
		{600, zerolog.InfoLevel},
	} {
		if got := accessLogLevel(tt.status); got != tt.want {
			t.Errorf("статус %d: уровень %s, ожидался %s", tt.status, got, tt.want)
		}
	}
}

func readRegressionEvents(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var events []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatal(err)
		}
		events = append(events, event)
	}
	return events
}
