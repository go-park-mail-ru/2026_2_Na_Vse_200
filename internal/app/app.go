package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"2026_2_Na_Vse_200/internal/config"
	"2026_2_Na_Vse_200/internal/repository/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Run(ctx context.Context, cfg config.Config) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	pool, err := connectPostgres(ctx, cfg.PostgreSQLDSN)
	if err != nil {
		return err
	}
	defer pool.Close()

	sessions := postgres.NewSessionRepo(pool)
	handler, err := newHandler(cfg, pool, sessions, logger)
	if err != nil {
		return fmt.Errorf("initialize media: %w", err)
	}
	server := newServer(cfg.Addr, handler)
	cleanupCtx, cancelCleanup := context.WithCancel(ctx)
	cleanupDone := make(chan struct{})
	go func() {
		defer close(cleanupDone)
		cleanupExpiredSessions(cleanupCtx, logger, sessions)
	}()
	defer func() {
		cancelCleanup()
		<-cleanupDone
	}()

	return serve(ctx, server, cfg.ShutdownTimeout, logger)
}

func connectPostgres(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if dsn == "" {
		return nil, errors.New("конфигурация: POSTGRES_DSN не задан")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("postgresql: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgresql: %w", err)
	}
	return pool, nil
}
