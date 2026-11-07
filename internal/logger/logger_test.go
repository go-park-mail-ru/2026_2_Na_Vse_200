package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestRequestIDRoundTrip(t *testing.T) {
	ctx := WithRequestID(context.Background(), "req-1")
	if got := RequestID(ctx); got != "req-1" {
		t.Fatalf("RequestID = %q", got)
	}
	if got := RequestID(context.Background()); got != "" {
		t.Fatalf("пустой контекст: %q", got)
	}
}

func TestFromContextAddsRequestID(t *testing.T) {
	var buf bytes.Buffer
	base := slog.New(slog.NewJSONHandler(&buf, nil))
	ctx := WithRequestID(context.Background(), "abc")

	FromContext(ctx, base).Info("проверка")

	var fields map[string]any
	if err := json.Unmarshal(buf.Bytes(), &fields); err != nil {
		t.Fatalf("json: %v", err)
	}
	if fields["request_id"] != "abc" {
		t.Fatalf("request_id = %v", fields["request_id"])
	}
	if fields["msg"] != "проверка" {
		t.Fatalf("msg = %v", fields["msg"])
	}
}
