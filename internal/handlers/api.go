// Package handlers содержит HTTP-обработчики сервиса и таблицу маршрутов.
package handlers

import (
	"net/http"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/auth"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/config"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository"
)

// Deps — зависимости обработчиков.
type Deps struct {
	Users    repository.UserRepositoryInterface
	Sessions repository.SessionRepositoryInterface
	Catalog  repository.CatalogRepositoryInterface
	Hasher   auth.Hasher
}

// API — набор обработчиков со своими зависимостями.
type API struct {
	cfg  *config.Config
	deps *Deps
}

// New создаёт набор обработчиков. Аргумент cfg задаёт настройки сервиса,
// deps — репозитории и хеширование паролей, которыми обработчики пользуются.
func New(cfg *config.Config, deps *Deps) *API {
	return &API{cfg: cfg, deps: deps}
}

// Routes возвращает таблицу маршрутов. Метод в шаблоне обязателен:
// по нему ServeMux сам отвечает 405 на неподходящий метод.
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
