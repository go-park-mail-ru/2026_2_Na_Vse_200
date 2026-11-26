package app

import (
	"context"
	"fmt"
	"net"
	"time"

	authv1 "2026_2_Na_Vse_200/auth-microservice/api/auth/v1"
	"2026_2_Na_Vse_200/auth-microservice/authrpc"
	"2026_2_Na_Vse_200/auth-microservice/internal/config"
	"2026_2_Na_Vse_200/auth-microservice/repository"
	"2026_2_Na_Vse_200/auth-microservice/sessions"
	"2026_2_Na_Vse_200/pkg/postgres"
	"2026_2_Na_Vse_200/pkg/rpcmetrics"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
)

func RunAuth(ctx context.Context, cfg config.Config, logger zerolog.Logger) error {
	ctx = logger.WithContext(ctx)
	pool, err := postgres.Connect(ctx, cfg.PostgreSQLDSN)
	if err != nil {
		return err
	}
	defer pool.Close()

	listener, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("запуск gRPC-сервера Auth: %w", err)
	}
	defer listener.Close()

	repo := repository.NewSessionRepo(pool)
	manager := sessions.NewSessionManager(repo, sessions.NewSessionID, time.Now, cfg.SessionTTL)
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
		cleanupInactiveSessions(cleanupCtx, logger, repo)
	}()
	defer func() {
		cancelCleanup()
		<-cleanupDone
	}()

	return metrics.RunWithServer(ctx, cfg.MetricsAddr, cfg.ShutdownTimeout, logger, func(ctx context.Context) error {
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
	conn, err := authv1.Dial(addr)
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
