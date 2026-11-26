package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

func New(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       60 * time.Second,
	}
}

func Serve(ctx context.Context, server *http.Server, timeout time.Duration, logger zerolog.Logger) error {
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info().Str("addr", server.Addr).Msg("запуск HTTP-сервера")
		err := server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serverErrors <- err
	}()

	select {
	case err := <-serverErrors:
		if err != nil {
			return fmt.Errorf("сервер остановлен с ошибкой: %w", err)
		}
		return nil
	case <-ctx.Done():
		logger.Info().Msg("получен сигнал остановки, завершаем текущие запросы")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			closeErr := server.Close()
			<-serverErrors
			return fmt.Errorf("остановка сервера: %w", errors.Join(err, closeErr))
		}
		if err := <-serverErrors; err != nil {
			return fmt.Errorf("сервер остановлен с ошибкой: %w", err)
		}
		logger.Info().Msg("сервер остановлен")
		return nil
	}
}
