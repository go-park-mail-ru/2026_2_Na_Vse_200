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
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/storage/memory"
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

	// До готовности слоя на PostgreSQL аккаунты живут в памяти процесса
	// и пропадают при перезапуске.
	users := memory.NewUserStorage()

	api := handlers.New(cfg, handlers.Deps{
		Users:  users,
		Hasher: auth.NewBcryptHasher(),
	})

	// Идентификатор запроса нужен логу и записи о панике, поэтому идёт первым;
	// recover — снаружи остальных обёрток.
	handler := middleware.Chain(
		api.Routes(),
		middleware.WithRequestID,
		middleware.WithRecover(logger),
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
