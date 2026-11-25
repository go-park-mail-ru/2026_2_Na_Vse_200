package app

import (
	"context"
	"fmt"
	"net"
	"time"

	authv1 "2026_2_Na_Vse_200/api/auth/v1"
	"2026_2_Na_Vse_200/internal/auth"
	"2026_2_Na_Vse_200/internal/authrpc"
	"2026_2_Na_Vse_200/internal/config"
	"2026_2_Na_Vse_200/internal/repository/postgres"
	"2026_2_Na_Vse_200/internal/rpcmetrics"
	authusecase "2026_2_Na_Vse_200/internal/usecase/auth"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
)

func RunAuth(ctx context.Context, cfg config.Config, logger zerolog.Logger) error {
	ctx = logger.WithContext(ctx)
	pool, err := connectPostgres(ctx, cfg.PostgreSQLDSN)
	if err != nil {
		return err
	}
	defer pool.Close()

	listener, err := net.Listen("tcp", cfg.AuthListenAddr)
	if err != nil {
		return fmt.Errorf("запуск gRPC-сервера Auth: %w", err)
	}
	defer listener.Close()

	repo := postgres.NewSessionRepo(pool)
	manager := authusecase.NewSessionManager(repo, auth.NewSessionID, time.Now, cfg.SessionTTL)
	metrics := rpcmetrics.New("music-auth")
	server := grpc.NewServer(grpc.UnaryInterceptor(metrics.Server(logger)))
	defer server.Stop()
	authv1.RegisterAuthServiceServer(server, authrpc.NewServer(manager))
	healthServer := health.NewServer()
	healthv1.RegisterHealthServer(server, healthServer)
	healthServer.SetServingStatus(authv1.AuthService_ServiceDesc.ServiceName, healthv1.HealthCheckResponse_SERVING)

	cleanupCtx, cancelCleanup := context.WithCancel(ctx)
	cleanupDone := make(chan struct{})
	go func() {
		defer close(cleanupDone)
		cleanupExpiredSessions(cleanupCtx, logger, repo)
	}()
	defer func() {
		cancelCleanup()
		<-cleanupDone
	}()

	return runWithMetrics(ctx, cfg.AuthMetricsAddr, metrics, cfg.ShutdownTimeout, logger, func(ctx context.Context) error {
		logger.Info().Str("addr", listener.Addr().String()).Msg("запуск gRPC-сервера Auth")
		return serveAuth(ctx, server, healthServer, listener, cfg.ShutdownTimeout)
	})
}

func serveAuth(
	ctx context.Context,
	server *grpc.Server,
	healthServer *health.Server,
	listener net.Listener,
	shutdownTimeout time.Duration,
) error {
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(listener)
	}()
	select {
	case err := <-serveErr:
		return fmt.Errorf("gRPC-сервер Auth: %w", err)
	case <-ctx.Done():
	}

	healthServer.Shutdown()
	done := make(chan struct{})
	go func() {
		defer close(done)
		server.GracefulStop()
	}()

	timer := time.NewTimer(shutdownTimeout)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
		server.Stop()
		<-done
	}
	return nil
}

func CheckAuth(ctx context.Context, addr string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	conn, err := dialAuth(addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	reply, err := healthv1.NewHealthClient(conn).Check(ctx, &healthv1.HealthCheckRequest{
		Service: authv1.AuthService_ServiceDesc.ServiceName,
	})
	if err != nil {
		return err
	}
	if reply.GetStatus() != healthv1.HealthCheckResponse_SERVING {
		return fmt.Errorf("Auth не готов: %s", reply.GetStatus())
	}
	return nil
}

func dialAuth(target string, options ...grpc.DialOption) (*grpc.ClientConn, error) {
	direct := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		// Auth's internal address must resolve inside the application network.
		grpc.WithNoProxy(),
		// Compose provides address records for Auth. Service settings are local.
		grpc.WithDisableServiceConfig(),
	}
	return grpc.NewClient(target, append(direct, options...)...)
}
