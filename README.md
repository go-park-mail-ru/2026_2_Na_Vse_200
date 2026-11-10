# На все 200

Бэкенд музыкального сервиса 2une. Go, PostgreSQL, pgx и стандартный HTTP-сервер `net/http`.

## Запуск

Нужен запущенный Docker Desktop в режиме Linux containers или Docker Engine с Compose. Из корня проекта:

```bash
docker compose up --build -d
```

API доступен на `http://localhost:8080`. Проверка сервера — [GET /health](http://localhost:8080/health).
Compose запускает PostgreSQL и применяет миграции перед запуском API. Для первого запуска `.env` не требуется; настройки можно изменить по образцу `.env.example`.

Логи и остановка:

```bash
docker compose logs -f server
docker compose down
```

Данные PostgreSQL сохраняются в volume. Подключение к базе с компьютера — `localhost:5433`.

## Проверка и сборка

Нужен Go 1.27.1 или новее.

```bash
go test ./...
go vet ./...
go build -o app ./cmd/app
```

Собранное приложение принимает команды `server` и `migrate up`. При запуске без Docker настройки задаются в окружении, включая `POSTGRES_DSN`; `.env` автоматически не загружается.

Интеграционные тесты PostgreSQL запускаются только с `TEST_POSTGRES_DSN`, указывающим на отдельную тестовую базу. Подсчёт покрытия без сгенерированных моков описан в [инструкции по тестам](tests/README.md).

## Разработка

- `internal/delivery` — HTTP-обработчики, маршруты и middleware.
- `internal/usecase` — бизнес-логика; `ports` — интерфейсы её зависимостей.
- `internal/repository/postgres` — SQL-запросы и работа с PostgreSQL.
- `db/migrations` — SQL-миграции.
- Моки интерфейсов генерируются через `go generate ./internal/usecase/ports`. В тестах используются Uber GoMock и sqlmock.

Подробнее: [конфигурация и структура](internal/README.md), [база данных и миграции](db/README.md), [тесты](tests/README.md).

## Команда

- [Федоров Федор](https://github.com/1ffedor)
- [Сайфетдинов Андрей](https://github.com/Andre1ka11)
- [Кузнецов Станислав](https://github.com/Stadmi)
- [Селибов Артём](https://github.com/BezFantasii)

## Ссылки

- [Фронтенд](https://github.com/frontend-park-mail-ru/2026_2_Na_Vse_200)
- [Figma](https://www.figma.com/design/a6FEt86x6uF36jjA64WcW8/MUSIC?node-id=0-1&t=Yrg1OBQLQ2NnfmIr-0)
- [Сайт](http://161.104.107.92/)

### Правила оформления Pull Requests

1. Ветка создается с названием `MUSIC-###`, где ### — номер задачи.
2. Название Pull Request'а соответствует названию задачи: `MUSIC-###: description`, где description — название задачи (что вы реализовали в этом Pull Request'е).
3. Для того, чтобы залить изменения в ветку `main`, нужен апрув от [Тимофея](https://t.me/TimofeyChichikin).
