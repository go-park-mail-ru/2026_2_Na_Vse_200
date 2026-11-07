package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"2026_2_Na_Vse_200/internal/auth"
	"2026_2_Na_Vse_200/internal/config"
	"2026_2_Na_Vse_200/internal/delivery/handlers"
	"2026_2_Na_Vse_200/internal/delivery/handlers/media"
	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"2026_2_Na_Vse_200/internal/repository/postgres"
	"2026_2_Na_Vse_200/internal/storage"
	artistusecase "2026_2_Na_Vse_200/internal/usecase/artist"
	authusecase "2026_2_Na_Vse_200/internal/usecase/auth"
	catalogusecase "2026_2_Na_Vse_200/internal/usecase/catalog"
	profileusecase "2026_2_Na_Vse_200/internal/usecase/profile"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

func newHandler(cfg config.Config, pool *pgxpool.Pool, sessions *postgres.SessionRepo, logger zerolog.Logger) (http.Handler, error) {
	blobs, err := storage.NewLocal(cfg.MediaRoot)
	if err != nil {
		return nil, err
	}

	users := postgres.NewUserRepo(pool)
	mediaRepo := postgres.NewMediaRepo(pool)
	api := handlers.New(&cfg, &handlers.Deps{
		Auth:    authusecase.New(users, sessions, auth.NewBcryptHasher(), auth.NewSessionID, time.Now, cfg.SessionTTL),
		Catalog: catalogusecase.New(postgres.NewCatalogRepo(pool)),
		Profile: profileusecase.NewProfileUsecase(users, users, mediaRepo, blobs),
		Artist:  artistusecase.NewArtistUsecase(postgres.NewArtistRepo(pool)),
		Media:   media.NewMediaHandler(mediaRepo, blobs),
	})
	return middleware.Chain(
		api.Routes(),
		middleware.WithRequestID,
		middleware.WithLogging(logger, "monolith/middleware"),
		middleware.WithRecover(),
		middleware.WithCORS(cfg.AllowedOrigin, cfg.AllowedMethods, cfg.AllowedHeaders),
		middleware.WithJSONErrors,
	), nil
}

func newServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func serve(ctx context.Context, server *http.Server, timeout time.Duration, logger zerolog.Logger) error {
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info().Str("addr", server.Addr).Str("handled_by", "monolith/server").Msg("запуск сервера")
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
