package handlers

import (
	"2026_2_Na_Vse_200/internal/delivery/handlers/media"
	"2026_2_Na_Vse_200/internal/delivery/handlers/profile"
	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"net/http"

	"2026_2_Na_Vse_200/internal/config"
	"2026_2_Na_Vse_200/internal/delivery/handlers/auth"
	"2026_2_Na_Vse_200/internal/delivery/handlers/health"
	"2026_2_Na_Vse_200/internal/delivery/handlers/home"
)

type Deps struct {
	Auth     auth.Usecase
	Sessions middleware.SessionByIDFunc
	Profile  profile.ProfileUsecase
	Media    *media.Handler
	Catalog  home.CatalogUsecase
	Artist   home.ArtistUsecase
}

type API struct {
	auth    *auth.Handler
	profile *profile.ProfileHandler
	media   *media.Handler
	protect middleware.Middleware
	home    *home.Handler
	artist  *home.ArtistHandler
}

func New(cfg *config.Config, deps *Deps) *API {
	return &API{
		auth:    auth.New(cfg, deps.Auth),
		profile: profile.NewProfileHandler(deps.Profile),
		media:   deps.Media,
		protect: middleware.WithAuth("session_id", deps.Sessions),
		home:    home.New(deps.Catalog),
		artist:  home.NewArtist(deps.Artist),
	}
}

func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health.Health)
	mux.HandleFunc("POST /api/v1/auth/signup", a.auth.Signup)
	mux.HandleFunc("POST /api/v1/auth/login", a.auth.Login)
	mux.HandleFunc("GET /api/v1/auth/me", a.auth.Me)
	mux.HandleFunc("POST /api/v1/auth/logout", a.auth.Logout)
	mux.HandleFunc("GET /api/v1/home", a.home.Home)
	mux.HandleFunc("GET /api/v1/artists/{id}", a.artist.Artist)
	if a.media != nil {
		mux.HandleFunc("GET /api/v1/media/{id}", a.media.Get)
		mux.HandleFunc("GET /api/v1/tracks/{id}/audio", a.media.Audio)
	}
	if a.profile != nil {
		mux.Handle("GET /api/v1/users/{id}", a.protect(http.HandlerFunc(a.profile.Get)))
		mux.Handle("PUT /api/v1/users/{id}", a.protect(http.HandlerFunc(a.profile.Update)))
		mux.Handle("POST /api/v1/users/{id}/avatar", a.protect(http.HandlerFunc(a.profile.UploadAvatar)))
	}
	return mux
}
