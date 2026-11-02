// Команда server поднимает HTTP API музыкального сервиса.
package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/auth"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/config"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/handlers"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("конфигурация: %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	if cfg.PostgreSQLDSN == "" {
		log.Fatalf("конфигурация: POSTGRES_DSN не задан")
	}

	pool, err := pgxpool.New(context.Background(), cfg.PostgreSQLDSN)
	if err != nil {
		log.Fatalf("postgresql: %v", err)
	}
	defer pool.Close()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = pool.Ping(pingCtx)
	pingCancel()
	if err != nil {
		log.Fatalf("postgresql: %v", err)
	}

	users := postgres.NewUserRepo(pool)

	sessions := postgres.NewSessionRepo(pool)
	catalog := postgres.NewCatalogRepo(pool)

	api := handlers.New(&cfg, &handlers.Deps{
		Users:    users,
		Sessions: sessions,
		Catalog:  catalog,
		Hasher:   auth.NewBcryptHasher(),
	})

	handler := middleware.Chain(
		api.Routes(),
		middleware.WithRecover(logger),
		middleware.WithRequestID,
		middleware.WithLogging(logger, "monolith/middleware"),
		middleware.WithCORS(cfg.AllowedOrigin, cfg.AllowedMethods, cfg.AllowedHeaders),
		middleware.WithJSONErrors,
	)

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
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

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("запуск сервера", slog.String("addr", cfg.Addr), slog.String("handled_by", "monolith/server"))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
			return
		}
		serverErrors <- nil
	}()

	select {
	case err := <-serverErrors:
		if err != nil {
			log.Fatalf("сервер остановлен с ошибкой: %v", err)
		}
	case <-ctx.Done():
		logger.Info("получен сигнал остановки, завершаем текущие запросы")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("остановка сервера: %v", err)
		}
		logger.Info("сервер остановлен")
	}
}

// cleanupExpiredSessions удаляет истёкшие сессии при запуске и раз в час.
// Проверка авторизации отклоняет их независимо от выполнения очистки.
func cleanupExpiredSessions(ctx context.Context, logger *slog.Logger, sessions *postgres.SessionRepo) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		cleanupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := sessions.DeleteExpired(cleanupCtx)
		cancel()
		if err != nil && ctx.Err() == nil {
			logger.Error("очистка истёкших сессий", slog.Any("error", err))
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
