package postgres

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"2026_2_Na_Vse_200/internal/logger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// QueryTracer пишет результат и длительность SQL-запроса без его параметров.
type QueryTracer struct {
	Log *slog.Logger
}

type queryKey struct{}

type queryInfo struct {
	started time.Time
	sql     string
}

func (t *QueryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, queryKey{}, queryInfo{started: time.Now(), sql: strings.Join(strings.Fields(data.SQL), " ")})
}

func (t *QueryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	query := ctx.Value(queryKey{}).(queryInfo)
	log := logger.FromContext(ctx, t.Log)
	attrs := []any{
		slog.String("sql", query.sql),
		slog.Float64("duration_ms", float64(time.Since(query.started).Microseconds())/1000),
		slog.Int64("rows", data.CommandTag.RowsAffected()),
	}
	if data.Err != nil && !errors.Is(data.Err, pgx.ErrNoRows) {
		// Detail у PostgreSQL может содержать значения полей, поэтому пишем только код.
		var pgErr *pgconn.PgError
		if errors.As(data.Err, &pgErr) {
			attrs = append(attrs, slog.String("sqlstate", pgErr.Code))
		} else {
			attrs = append(attrs, slog.String("error", "query failed"))
		}
		log.ErrorContext(ctx, "запрос к БД завершился ошибкой", attrs...)
		return
	}
	log.InfoContext(ctx, "запрос к БД выполнен", attrs...)
}
