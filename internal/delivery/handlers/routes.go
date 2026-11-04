package handlers

import (
	"net/http"

	"2026_2_Na_Vse_200/internal/config"
	"2026_2_Na_Vse_200/internal/delivery/handlers/artist"
	"2026_2_Na_Vse_200/internal/delivery/handlers/auth"
	"2026_2_Na_Vse_200/internal/delivery/handlers/health"
	"2026_2_Na_Vse_200/internal/delivery/handlers/home"
	"2026_2_Na_Vse_200/internal/delivery/handlers/profile"
	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"2026_2_Na_Vse_200/internal/usecase/ports"
)

const sessionCookieName = "session_id"

type Deps struct {
	Auth     auth.Usecase
	Catalog  home.CatalogUsecase
	Profile  profile.Usecase
	Artist   artist.Usecase
	Sessions ports.SessionRepository
}

type API struct {
	auth     *auth.Handler
	home     *home.Handler
	profile  *profile.Handler
	artist   *artist.Handler
	sessions ports.SessionRepository
}

func New(cfg *config.Config, deps *Deps) *API {
	return &API{
		auth:     auth.New(cfg, deps.Auth),
		home:     home.New(deps.Catalog),
		profile:  profile.New(deps.Profile),
		artist:   artist.New(deps.Artist),
		sessions: deps.Sessions,
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
	mux.Handle("GET /api/v1/users/{id}", middleware.WithAuth(sessionCookieName, a.sessions)(http.HandlerFunc(a.profile.Get)))
	mux.HandleFunc("GET /api/v1/artists/{id}", a.artist.Get)
	return mux
}
