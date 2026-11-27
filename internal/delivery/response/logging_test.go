package response

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
)

func TestResponseErrorsKeepRequestLogger(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		data      any
		writeFail bool
		status    int
	}{
		{"marshal", "response.marshal", make(chan int), false, http.StatusInternalServerError},
		{"write", "response.write", map[string]string{"status": "ok"}, true, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := zerolog.New(&buf).With().Str("request_id", "test-request").Logger()
			ctx := logger.WithContext(context.Background())
			recorder := httptest.NewRecorder()
			var writer http.ResponseWriter = recorder
			if tt.writeFail {
				writer = failingWriter{ResponseWriter: recorder}
			}
			WriteJSON(ctx, writer, http.StatusOK, tt.data)
			if recorder.Code != tt.status {
				t.Fatalf("status=%d, want %d", recorder.Code, tt.status)
			}
			var event map[string]any
			if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
				t.Fatal(err)
			}
			if event["request_id"] != "test-request" || event["operation"] != tt.operation || event["level"] != "error" || event["error"] == nil {
				t.Fatalf("incomplete error event: %v", event)
			}
		})
	}
}

type failingWriter struct{ http.ResponseWriter }

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("connection closed") }
