package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"2026_2_Na_Vse_200/internal/logger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// TestQueryTracer проверяет поля лога и отсутствие секретов.
func TestQueryTracer(t *testing.T) {
	for _, tc := range []struct {
		name, level string
		err         error
	}{
		{"успех", "INFO", nil},
		{"нет строк", "INFO", pgx.ErrNoRows},
		{"ошибка PostgreSQL", "ERROR", &pgconn.PgError{Code: "23505", Detail: "secret-password"}},
		{"ошибка соединения", "ERROR", errors.New("connection failed: secret-password")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			tracer := &QueryTracer{Log: slog.New(slog.NewJSONHandler(&buf, nil))}
			ctx := logger.WithRequestID(context.Background(), "request-1")
			ctx = tracer.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{
				SQL:  "SELECT id\n FROM account WHERE email = $1",
				Args: []any{"secret-email", "secret-password", "secret-session"},
			})
			tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{CommandTag: pgconn.NewCommandTag("SELECT 1"), Err: tc.err})
			var entry struct {
				Level     string  `json:"level"`
				RequestID string  `json:"request_id"`
				SQL       string  `json:"sql"`
				Rows      int     `json:"rows"`
				Duration  float64 `json:"duration_ms"`
				SQLState  string  `json:"sqlstate"`
			}
			if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
				t.Fatal(err)
			}
			if entry.Level != tc.level || entry.RequestID != "request-1" || entry.Rows != 1 || entry.Duration < 0 {
				t.Fatalf("неверная запись лога: %s", buf.String())
			}
			if entry.SQL != "SELECT id FROM account WHERE email = $1" {
				t.Errorf("SQL = %q", entry.SQL)
			}
			if tc.name == "ошибка PostgreSQL" && entry.SQLState != "23505" {
				t.Errorf("sqlstate = %q", entry.SQLState)
			}
			if strings.Contains(buf.String(), "secret-") {
				t.Fatalf("в лог попали секретные данные: %s", buf.String())
			}
		})
	}
}
