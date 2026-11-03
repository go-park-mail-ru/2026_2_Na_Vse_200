package migration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

const (
	connectTimeout   = 5 * time.Second
	migrationTimeout = time.Minute
	lockTimeout      = 15 * time.Second
)

func connectDB(ctx context.Context, dsn string) (*sql.DB, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("POSTGRES_DSN is not set")
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {

		return nil, errors.New("POSTGRES_DSN has invalid syntax")
	}
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = connectTimeout
	}
	if cfg.RuntimeParams == nil {
		cfg.RuntimeParams = make(map[string]string)
	}
	cfg.RuntimeParams["statement_timeout"] = fmt.Sprint(migrationTimeout.Milliseconds())
	cfg.RuntimeParams["lock_timeout"] = fmt.Sprint(lockTimeout.Milliseconds())
	db := stdlib.OpenDB(*cfg)
	pingCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		return nil, errors.Join(fmt.Errorf("connect to PostgreSQL: %w", err), db.Close())
	}
	return db, nil
}
