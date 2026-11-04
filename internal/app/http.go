package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"2026_2_Na_Vse_200/internal/auth"
	"2026_2_Na_Vse_200/internal/config"
	"2026_2_Na_Vse_200/internal/delivery/handlers"
	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"2026_2_Na_Vse_200/internal/repository/postgres"
	artistusecase "2026_2_Na_Vse_200/internal/usecase/artist"
	authusecase "2026_2_Na_Vse_200/internal/usecase/auth"
	catalogusecase "2026_2_Na_Vse_200/internal/usecase/catalog"
	profileusecase "2026_2_Na_Vse_200/internal/usecase/profile"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newHandler(cfg config.Config, pool *pgxpool.Pool, sessions *postgres.SessionRepo, logger *slog.Logger) http.Handler {
	users := postgres.NewUserRepo(pool)
	api := handlers.New(&cfg, &handlers.Deps{
		Auth:     authusecase.New(users, sessions, auth.NewBcryptHasher(), auth.NewSessionID, time.Now, cfg.SessionTTL),
		Catalog:  catalogusecase.New(postgres.NewCatalogRepo(pool)),
		Profile:  profileusecase.New(users),
		Artist:   artistusecase.New(postgres.NewArtistRepo(pool)),
		Sessions: sessions,
	})
	return middleware.Chain(
		api.Routes(),
		middleware.WithRecover(logger),
		middleware.WithRequestID,
		middleware.WithLogging(logger, "monolith/middleware"),
		middleware.WithCORS(cfg.AllowedOrigin, cfg.AllowedMethods, cfg.AllowedHeaders),
		middleware.WithJSONErrors,
	)
}

func newServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func serve(ctx context.Context, server *http.Server, timeout time.Duration, logger *slog.Logger) error {
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("запуск сервера", slog.String("addr", server.Addr), slog.String("handled_by", "monolith/server"))
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
		logger.Info("получен сигнал остановки, завершаем текущие запросы")
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
		logger.Info("сервер остановлен")
		return nil
	}
}
