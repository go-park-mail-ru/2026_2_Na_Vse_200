package postgres

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPoolAcquireTimeout(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set TEST_POSTGRES_DSN to test pool acquisition")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns, cfg.MinConns, cfg.MinIdleConns = 1, 0, 0
	cfg.ConnConfig.Tracer = &QueryTracer{Log: slog.New(slog.DiscardHandler), AcquireTimeout: 250 * time.Millisecond}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var value int
	started := time.Now()
	err = pool.QueryRow(ctx, "SELECT 1").Scan(&value)
	conn.Release()
	if !errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
		t.Fatalf("ожидался таймаут пула, получено %v (контекст запроса: %v)", err, ctx.Err())
	}
	stat := pool.Stat()
	if stat.CanceledAcquireCount() != 1 {
		t.Fatalf("отменённых ожиданий: %d, ожидалось 1", stat.CanceledAcquireCount())
	}
	t.Logf("ожидание: %s, отменённых получений соединения: %d", time.Since(started), stat.CanceledAcquireCount())

	if err := pool.QueryRow(ctx, "SELECT 1 FROM pg_sleep(0.4)").Scan(&value); err != nil || value != 1 {
		t.Fatalf("запрос после освобождения соединения: значение %d, ошибка %v", value, err)
	}
}
