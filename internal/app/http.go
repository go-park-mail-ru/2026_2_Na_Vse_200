package app

import (
	"net/http"

	"2026_2_Na_Vse_200/auth-microservice/authrpc"
	"2026_2_Na_Vse_200/internal/auth"
	"2026_2_Na_Vse_200/internal/config"
	"2026_2_Na_Vse_200/internal/delivery/handlers"
	"2026_2_Na_Vse_200/internal/delivery/handlers/media"
	"2026_2_Na_Vse_200/internal/delivery/handlers/playlist"
	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"2026_2_Na_Vse_200/internal/repository/postgres"
	"2026_2_Na_Vse_200/internal/storage"
	accountusecase "2026_2_Na_Vse_200/internal/usecase/account"
	artistusecase "2026_2_Na_Vse_200/internal/usecase/artist"
	authusecase "2026_2_Na_Vse_200/internal/usecase/auth"
	catalogusecase "2026_2_Na_Vse_200/internal/usecase/catalog"
	profileusecase "2026_2_Na_Vse_200/internal/usecase/profile"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

func newHandler(
	cfg config.Config,
	pool *pgxpool.Pool,
	sessions *authrpc.Client,
	logger zerolog.Logger,
	metrics *middleware.HTTPMetrics,
) (http.Handler, error) {
	blobs, err := storage.NewLocal(cfg.MediaRoot)
	if err != nil {
		return nil, err
	}

	users := postgres.NewUserRepo(pool)
	mediaRepo := postgres.NewMediaRepo(pool)
	catalog := postgres.NewCatalogRepo(pool)
	tracks := postgres.NewTrackRepo(pool)
	api := handlers.New(&cfg, &handlers.Deps{
		Favorites:          tracks,
		Music:              catalog,
		Playlists:          playlist.New(postgres.NewPlaylistRepo(pool, blobs), blobs),
		Account:            accountusecase.NewAccountUsecase(users, users, auth.NewBcryptHasher()),
		Auth:               authusecase.NewAuthUsecase(users, sessions, auth.NewBcryptHasher()),
		CheckProfileAccess: sessions.CheckProfileAccess,
		Catalog:            catalogusecase.New(catalog),
		Profile:            profileusecase.NewProfileUsecase(users, users, blobs),
		Artist:             artistusecase.NewArtistUsecase(postgres.NewArtistRepo(pool)),
		Media:              media.NewMediaHandler(mediaRepo, blobs, tracks),
	})
	return metrics.Wrap(
		api.Routes(),
		middleware.WithRequestID,
		middleware.WithLogging(logger, "http/middleware"),
		middleware.WithRecover(),
		middleware.WithCORS(cfg.AllowedOrigin, cfg.AllowedMethods, cfg.AllowedHeaders),
		middleware.WithCSRF,
		middleware.WithJSONErrors,
	), nil
}
