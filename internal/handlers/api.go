// Package handlers содержит HTTP-обработчики сервиса и таблицу маршрутов.
package handlers

import (
	"log"
	"net/http"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/apimessage"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/auth"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/config"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/pkg/response"
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
	cfg  config.Config
	deps Deps
}

// writeInternalError логирует причину и отдаёт клиенту общий текст без деталей.
func writeInternalError(w http.ResponseWriter, context string, err error) {
	log.Printf("%s: %v", context, err)
	response.WriteError(w, http.StatusInternalServerError,
		apimessage.CodeInternal, apimessage.MsgInternal)
}

// New создаёт набор обработчиков. Аргумент cfg задаёт настройки сервиса,
// deps — репозитории и хеширование паролей, которыми обработчики пользуются.
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
