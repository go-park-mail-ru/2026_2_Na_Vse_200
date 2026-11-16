package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog"
)

type dbOperationKey struct{}
type queryTraceKey struct{}

type queryTrace struct {
	startedAt time.Time
	operation string
	statement string
}

func withDBOperation(ctx context.Context, operation string) context.Context {
	return context.WithValue(ctx, dbOperationKey{}, operation)
}

// QueryTracer пишет события pgx через логгер из контекста запроса.
type QueryTracer struct{}

var _ pgx.QueryTracer = QueryTracer{}

func (QueryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	log := zerolog.Ctx(ctx)
	if log.GetLevel() > zerolog.DebugLevel || zerolog.GlobalLevel() > zerolog.DebugLevel {
		return ctx
	}
	operation, _ := ctx.Value(dbOperationKey{}).(string)
	if operation == "" {
		operation = "postgres.query"
	}
	trace := queryTrace{
		startedAt: time.Now(),
		operation: operation,
		statement: statementKind(data.SQL),
	}
	log.Debug().
		Str("layer", "repository").
		Str("operation", trace.operation).
		Str("db_statement", trace.statement).
		Str("phase", "start").
		Msg("начало запроса к PostgreSQL")
	return context.WithValue(ctx, queryTraceKey{}, trace)
}

func (QueryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	trace, ok := ctx.Value(queryTraceKey{}).(queryTrace)
	if !ok {
		return
	}
	event := zerolog.Ctx(ctx).Debug().
		Str("layer", "repository").
		Str("operation", trace.operation).
		Str("db_statement", trace.statement).
		Str("phase", "finish").
		Float64("duration_ms", float64(time.Since(trace.startedAt))/float64(time.Millisecond))

	switch {
	case data.Err == nil:
		event.Str("outcome", "ok").Int64("rows_affected", data.CommandTag.RowsAffected())
	case errors.Is(data.Err, context.Canceled):
		event.Str("outcome", "canceled")
	case errors.Is(data.Err, context.DeadlineExceeded):
		event.Str("outcome", "deadline_exceeded")
	default:
		event.Str("outcome", "error")
		var pgErr *pgconn.PgError
		if errors.As(data.Err, &pgErr) {
			event.Str("sqlstate", pgErr.Code)
		}
	}
	event.Msg("завершён запрос к PostgreSQL")
}

func statementKind(sql string) string {
	fields := strings.Fields(sql)
	if len(fields) > 0 {
		switch keyword := strings.ToUpper(fields[0]); keyword {
		case "SELECT", "INSERT", "UPDATE", "DELETE", "WITH", "BEGIN", "COMMIT", "ROLLBACK":
			return keyword
		}
	}
	return "OTHER"
}
