package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"2026_2_Na_Vse_200/internal/authrpc"
	"2026_2_Na_Vse_200/internal/config"
	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"2026_2_Na_Vse_200/internal/repository/postgres"
	"2026_2_Na_Vse_200/internal/rpcmetrics"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

func Run(ctx context.Context, cfg config.Config, logger zerolog.Logger) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	ctx = logger.WithContext(ctx)
	pool, err := connectPostgres(ctx, cfg.PostgreSQLDSN)
	if err != nil {
		return err
	}
	defer pool.Close()

	metrics := rpcmetrics.New("music-api")
	conn, err := dialAuth(cfg.AuthTarget,
		grpc.WithDisableRetry(),
		grpc.WithChainUnaryInterceptor(withAuthTimeout, metrics.Client),
	)
	if err != nil {
		return fmt.Errorf("подключение gRPC-клиента Auth: %w", err)
	}
	defer conn.Close()

	httpMetrics := middleware.NewHTTPMetrics("monolith")
	handler, err := newHandler(cfg, pool, authrpc.NewClient(conn), logger, httpMetrics)
	if err != nil {
		return err
	}
	server := newServer(cfg.Addr, handler)
	if cfg.MetricsAddr != "" {
		stopMetrics, err := startMetrics(cfg.MetricsAddr, combinedMetrics(httpMetrics, metrics), logger)
		if err != nil {
			return err
		}
		defer stopMetrics()
	}
	return serve(ctx, server, cfg.ShutdownTimeout, logger)
}

func withAuthTimeout(
	ctx context.Context,
	method string,
	req, reply any,
	conn *grpc.ClientConn,
	invoke grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	const authRPCTimeout = 3 * time.Second
	ctx, cancel := context.WithTimeout(ctx, authRPCTimeout)
	defer cancel()
	return invoke(ctx, method, req, reply, conn, opts...)
}

func connectPostgres(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if dsn == "" {
		return nil, errors.New("конфигурация: POSTGRES_DSN не задан")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgresql: %w", err)
	}
	cfg.ConnConfig.Tracer = postgres.QueryTracer{}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
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
