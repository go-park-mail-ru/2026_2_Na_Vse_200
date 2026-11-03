package handlers

import (
	"net/http"

	"2026_2_Na_Vse_200/internal/config"
	"2026_2_Na_Vse_200/internal/delivery/handlers/auth"
	"2026_2_Na_Vse_200/internal/delivery/handlers/health"
	"2026_2_Na_Vse_200/internal/delivery/handlers/home"
)

type Deps struct {
	Auth    auth.Usecase
	Catalog home.CatalogUsecase
}

type API struct {
	auth *auth.Handler
	home *home.Handler
}

func New(cfg *config.Config, deps *Deps) *API {
	return &API{
		auth: auth.New(cfg, deps.Auth),
		home: home.New(deps.Catalog),
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
	return mux
}
