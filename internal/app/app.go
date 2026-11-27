package app

import (
	"context"
	"fmt"
	"time"

	authv1 "2026_2_Na_Vse_200/auth-microservice/api/auth/v1"
	"2026_2_Na_Vse_200/internal/config"
	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"2026_2_Na_Vse_200/pkg/httpserver"
	"2026_2_Na_Vse_200/pkg/postgres"
	"2026_2_Na_Vse_200/pkg/rpcmetrics"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

func Run(ctx context.Context, cfg config.Config, logger zerolog.Logger) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	ctx = logger.WithContext(ctx)
	pool, err := postgres.Connect(ctx, cfg.PostgreSQLDSN)
	if err != nil {
		return err
	}
	defer pool.Close()

	metrics := rpcmetrics.New("music-api")
	conn, err := authv1.Dial(cfg.AuthTarget,
		grpc.WithDisableRetry(),
		grpc.WithChainUnaryInterceptor(withAuthTimeout, metrics.Client),
	)
	if err != nil {
		return fmt.Errorf("подключение gRPC-клиента Auth: %w", err)
	}
	defer conn.Close()

	httpMetrics := middleware.NewHTTPMetrics("monolith")
	handler, err := newHandler(cfg, pool, newAuthClient(conn), logger, httpMetrics)
	if err != nil {
		return err
	}
	server := httpserver.New(cfg.Addr, handler)
	if cfg.MetricsAddr != "" {
		stopMetrics, err := startMetrics(cfg.MetricsAddr, combinedMetrics(httpMetrics, metrics), logger)
		if err != nil {
			return err
		}
		defer stopMetrics()
	}
	return httpserver.Serve(ctx, server, cfg.ShutdownTimeout, logger)
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
