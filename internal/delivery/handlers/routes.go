package handlers

import (
	"net/http"

	"2026_2_Na_Vse_200/internal/config"
	"2026_2_Na_Vse_200/internal/delivery/handlers/artist"
	"2026_2_Na_Vse_200/internal/delivery/handlers/auth"
	"2026_2_Na_Vse_200/internal/delivery/handlers/health"
	"2026_2_Na_Vse_200/internal/delivery/handlers/home"
	"2026_2_Na_Vse_200/internal/delivery/handlers/media"
	"2026_2_Na_Vse_200/internal/delivery/handlers/profile"
	"2026_2_Na_Vse_200/internal/delivery/middleware"
)

const sessionCookieName = "session_id"

type Deps struct {
	Auth    auth.Usecase
	Catalog home.CatalogUsecase
	Profile profile.ProfileUsecase
	Artist  artist.ArtistUsecase
	Media   *media.MediaHandler
}

type API struct {
	auth    auth.Usecase
	authH   *auth.Handler
	home    *home.Handler
	profile *profile.ProfileHandler
	artist  *artist.ArtistHandler
	media   *media.MediaHandler
}

func New(cfg *config.Config, deps *Deps) *API {
	return &API{
		auth:    deps.Auth,
		authH:   auth.New(cfg, deps.Auth),
		home:    home.New(deps.Catalog),
		profile: profile.NewProfileHandler(deps.Profile),
		artist:  artist.NewArtistHandler(deps.Artist),
		media:   deps.Media,
	}
}

func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health.Health)
	mux.HandleFunc("POST /api/v1/auth/signup", a.authH.Signup)
	mux.HandleFunc("POST /api/v1/auth/login", a.authH.Login)
	mux.HandleFunc("GET /api/v1/auth/me", a.authH.Me)
	mux.HandleFunc("POST /api/v1/auth/logout", a.authH.Logout)
	mux.HandleFunc("GET /api/v1/home", a.home.Home)

	if a.auth != nil && a.profile != nil {
		authUser := middleware.WithAuth(sessionCookieName, a.auth.SessionByID)
		mux.Handle("GET /api/v1/users/{id}", authUser(http.HandlerFunc(a.profile.Get)))
		mux.Handle("PUT /api/v1/users/{id}", authUser(http.HandlerFunc(a.profile.Update)))
		mux.Handle("POST /api/v1/users/{id}/avatar", authUser(http.HandlerFunc(a.profile.UploadAvatar)))
	}
	if a.artist != nil {
		mux.HandleFunc("GET /api/v1/artists/{id}", a.artist.Get)
	}
	if a.media != nil {
		mux.HandleFunc("GET /api/v1/media/{id}", a.media.Get)
	}
	return mux
}
