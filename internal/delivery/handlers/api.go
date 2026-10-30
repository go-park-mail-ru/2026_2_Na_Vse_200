// Package handlers adapts HTTP to application scenarios.
package handlers

import (
	"context"
	"net/http"

	"2026_2_Na_Vse_200/internal/config"
	"2026_2_Na_Vse_200/internal/models"
	authusecase "2026_2_Na_Vse_200/internal/usecase/auth"
	catalogusecase "2026_2_Na_Vse_200/internal/usecase/catalog"
)

type AuthUsecase interface {
	Signup(context.Context, authusecase.SignupInput) (authusecase.Result, error)
	Login(context.Context, authusecase.LoginInput) (authusecase.Result, error)
	Me(context.Context, models.SessionID) (models.User, error)
	Logout(context.Context, models.SessionID) error
}
type CatalogUsecase interface {
	Home(context.Context) (catalogusecase.Home, error)
}
type Deps struct {
	Auth    AuthUsecase
	Catalog CatalogUsecase
}
type API struct {
	cfg  *config.Config
	deps *Deps
}

func New(cfg *config.Config, deps *Deps) *API { return &API{cfg: cfg, deps: deps} }
func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.Health)
	mux.HandleFunc("POST /api/v1/auth/signup", a.Signup)
	mux.HandleFunc("POST /api/v1/auth/login", a.Login)
	mux.HandleFunc("GET /api/v1/auth/me", a.Me)
	mux.HandleFunc("POST /api/v1/auth/logout", a.Logout)
	mux.HandleFunc("GET /api/v1/home", a.Home)
	return mux
}
