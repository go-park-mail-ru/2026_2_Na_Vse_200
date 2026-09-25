package handlers

import (
	"net/http"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/auth"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/config"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/storage"
)

// Deps — зависимости обработчиков. Интерфейсы, а не конкретные типы:
// в тестах подставляются свои реализации.
type Deps struct {
	Users    storage.UserStorageInterface
	Sessions storage.SessionStorageInterface
	Catalog  storage.CatalogStorageInterface
	Hasher   auth.Hasher
}

// API — набор обработчиков со своими зависимостями.
type API struct {
	cfg  config.Config
	deps Deps
}

// New собирает API.
func New(cfg config.Config, deps Deps) *API {
	return &API{cfg: cfg, deps: deps}
}

// Routes возвращает таблицу маршрутов. Метод в шаблоне обязателен:
// по нему ServeMux сам отвечает 405 на неподходящий метод.
func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", a.Health)
	mux.HandleFunc("POST /api/v1/auth/signup", a.Signup)

	return mux
}
