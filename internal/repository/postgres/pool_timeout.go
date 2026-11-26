package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type acquireCancelKey struct{}

var _ pgxpool.AcquireTracer = (*QueryTracer)(nil)

// TraceAcquireStart ограничивает ожидание соединения отдельно от выполнения SQL.
func (t *QueryTracer) TraceAcquireStart(ctx context.Context, _ *pgxpool.Pool, _ pgxpool.TraceAcquireStartData) context.Context {
	if t.AcquireTimeout <= 0 {
		return ctx
	}
	ctx, cancel := context.WithTimeout(ctx, t.AcquireTimeout)
	return context.WithValue(ctx, acquireCancelKey{}, cancel)
}

func (t *QueryTracer) TraceAcquireEnd(ctx context.Context, _ *pgxpool.Pool, _ pgxpool.TraceAcquireEndData) {
	if cancel, ok := ctx.Value(acquireCancelKey{}).(context.CancelFunc); ok {
		cancel()
	}
}
