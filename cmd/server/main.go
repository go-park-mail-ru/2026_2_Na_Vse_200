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

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/auth"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/config"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/handlers"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository/memory"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository/postgres"
)

// component попадает в лог полем handled_by.
const component = "monolith/middleware"

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("конфигурация: %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Аккаунты в PostgreSQL. Сессии остаются в памяти: в схеме БД их нет,
	// при перезапуске сервера они пропадают, аккаунты сохраняются.
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
		pool.Close()
		log.Fatalf("postgresql: %v", err)
	}

	users := postgres.NewUserRepo(pool)
	sessions := memory.NewSessionRepo()

	api := handlers.New(&cfg, &handlers.Deps{
		Users:    users,
		Sessions: sessions,
		Hasher:   auth.NewBcryptHasher(),
	})

	// Recover идёт первым, то есть становится самой внешней обёрткой: паника
	// в любой из остальных тоже должна попасть в лог, а не уронить процесс.
	handler := middleware.Chain(
		api.Routes(),
		middleware.WithRecover(logger),
		middleware.WithRequestID,
		middleware.WithLogging(logger, component),
		middleware.WithCORS(cfg.AllowedOrigin),
		middleware.WithJSONErrors,
	)

	// Таймауты рассчитаны на JSON-API: самая долгая операция — проверка пароля,
	// около 60 мс. IdleTimeout больше остальных: это keep-alive между запросами.
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

	// ListenAndServe блокирует, поэтому ждём сигнал остановки в главной горутине.
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("сервер запущен", slog.String("addr", cfg.Addr), slog.String("component", component))
		// После Shutdown возвращается ErrServerClosed — это штатное завершение.
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
