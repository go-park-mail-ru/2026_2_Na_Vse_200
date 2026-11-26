package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog"
)

func TestQueryTracerCorrelatesEventsWithoutSQLOrArguments(t *testing.T) {
	for _, tt := range []struct {
		name     string
		err      error
		outcome  string
		sqlstate string
	}{
		{name: "success", outcome: "ok"},
		{name: "no rows", err: pgx.ErrNoRows, outcome: "ok"},
		{name: "connection error", err: errors.New("private-error-detail"), outcome: "error"},
		{name: "postgres error", err: &pgconn.PgError{Code: "23505", Detail: "private-error-detail"}, outcome: "error", sqlstate: "23505"},
		{name: "canceled", err: context.Canceled, outcome: "canceled"},
		{name: "timeout", err: context.DeadlineExceeded, outcome: "deadline_exceeded"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			log := zerolog.New(&output).Level(zerolog.DebugLevel).With().Str("request_id", "request-42").Logger()
			ctx := WithDBOperation(log.WithContext(context.Background()), "users.get_by_email")
			tracer := QueryTracer{}
			traceCtx := tracer.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{
				SQL:  " SELECT private_column FROM account WHERE email = $1",
				Args: []any{"private-argument"},
			})
			tracer.TraceQueryEnd(traceCtx, nil, pgx.TraceQueryEndData{
				CommandTag: pgconn.NewCommandTag("SELECT 1"),
				Err:        tt.err,
			})

			for _, secret := range []string{"private_column", "private-argument", "private-error-detail"} {
				if strings.Contains(output.String(), secret) {
					t.Errorf("в лог попало значение %q", secret)
				}
			}
			lines := bytes.Split(bytes.TrimSpace(output.Bytes()), []byte("\n"))
			if len(lines) != 2 {
				t.Fatalf("ожидались события начала и завершения: %s", output.String())
			}
			for i, phase := range []string{"start", "finish"} {
				var event map[string]any
				if err := json.Unmarshal(lines[i], &event); err != nil {
					t.Fatal(err)
				}
				if event["request_id"] != "request-42" || event["layer"] != "repository" ||
					event["operation"] != "users.get_by_email" || event["db_statement"] != "SELECT" ||
					event["phase"] != phase || event["level"] != "debug" {
					t.Fatalf("неверное событие %s: %v", phase, event)
				}
				if phase == "finish" {
					duration, ok := event["duration_ms"].(float64)
					sqlstate, _ := event["sqlstate"].(string)
					if !ok || duration < 0 || event["outcome"] != tt.outcome || sqlstate != tt.sqlstate {
						t.Errorf("неверный результат запроса: %v", event)
					}
					if tt.err == nil && event["rows_affected"] != float64(1) {
						t.Errorf("rows_affected = %v, ожидалось 1", event["rows_affected"])
					}
					if errors.Is(tt.err, pgx.ErrNoRows) && event["rows_affected"] != float64(0) {
						t.Errorf("rows_affected = %v, ожидалось 0", event["rows_affected"])
					}
				}
			}
		})
	}
}

func TestQueryTracerSkipsInfoLevel(t *testing.T) {
	var output bytes.Buffer
	log := zerolog.New(&output).Level(zerolog.InfoLevel)
	ctx := log.WithContext(context.Background())
	tracer := QueryTracer{}
	traceCtx := tracer.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: "SELECT 1"})
	tracer.TraceQueryEnd(traceCtx, nil, pgx.TraceQueryEndData{})
	if traceCtx != ctx || output.Len() != 0 {
		t.Fatal("трассировка запросов должна быть отключена на уровне info")
	}
}
