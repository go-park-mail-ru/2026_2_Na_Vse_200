package handlers

import (
	"net/http"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/config"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/storage"
)

// Deps — хранилища, которыми пользуются обработчики.
//
// Поля объявлены интерфейсами из пакета storage, а не конкретными типами:
// в main подставляется рабочая реализация, в тестах — подставная, и обработчики
// разницы не видят. Пока не заполняются: понадобятся регистрации, входу и главной.
type Deps struct {
	Users    storage.UserStorageInterface
	Sessions storage.SessionStorageInterface
	Catalog  storage.CatalogStorageInterface
}

// API — набор обработчиков со своими зависимостями.
// Зависимости лежат полями структуры, а не в глобальных переменных: глобальное
// состояние мешает тестам и прячет, чем на самом деле пользуется обработчик.
type API struct {
	cfg  config.Config
	deps Deps
}

// New собирает API.
func New(cfg config.Config, deps Deps) *API {
	return &API{cfg: cfg, deps: deps}
}

// Routes возвращает таблицу маршрутов сервиса.
//
// Метод указывается прямо в шаблоне ("GET /health"): так ServeMux сам отвечает
// 405 на неподходящий метод, а не отдаёт запрос обработчику. Шаблон без метода
// ловил бы все методы разом.
//
// Оставшиеся ручки контракта — регистрация, вход, текущий пользователь, выход
// и главная — добавляются сюда по одной строке каждая.
func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", a.Health)

	return mux
}
