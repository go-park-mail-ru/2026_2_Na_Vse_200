# 2026_2_Na_Vse_200
Backend репозиторий команды На все 200 с проектом Spotify/Яндекс Музыка

### Участники команды
 1. [Федоров Федор](https://github.com/1ffedor)
 2. [Сайфетдинов Андрей](https://github.com/Andre1ka11)
 3. [Кузнецов Станислав](https://github.com/Stadmi)
 4. [Селибов Артём](https://github.com/BezFantasii)

### Запуск

Нужен только Go (версия из `go.mod`), внешних зависимостей нет.

```
go run ./cmd/server
```

Сервер поднимается на http://localhost:8080. Проверка:

```
curl http://localhost:8080/health      # {"status":"ok"}
```

Остановка — Ctrl+C: сервер дожидается текущих запросов и только потом завершается.

Контракт API (какие есть ручки, что отвечают, формат ошибок) — в [docs/api.md](docs/api.md).

### Конфигурация

Всё читается из переменных окружения, значения по умолчанию рассчитаны на локальный запуск —
настраивать ничего не нужно. Секреты в репозиторий не коммитим, `.env` в `.gitignore`.

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `APP_ADDR` | `:8080` | адрес, который слушает сервер |
| `APP_ALLOWED_ORIGINS` | пусто | адреса фронтенда для CORS через запятую; пусто — CORS выключен |
| `APP_COOKIE_SECURE` | `false` | флаг `Secure` у сессионной cookie; на стенде с HTTPS — `true` |
| `APP_SESSION_TTL` | `24h` | срок жизни сессии |
| `APP_SHUTDOWN_TIMEOUT` | `10s` | сколько ждём завершения запросов при остановке |
| `DB_DSN` | пусто | строка подключения к PostgreSQL (используется с BE-03) |

Пример запуска с фронтом на соседнем порту:

```
APP_ALLOWED_ORIGINS=http://localhost:3000 go run ./cmd/server
```

### Проверки перед PR

```
go build ./... && go vet ./...
gofmt -l .
go test ./... -cover
```

### Внешние ссылки - TODO
 - [Фронтенд проекта](https://github.com/frontend-park-mail-ru/2026_2_Na_Vse_200)
 - [Figma](https://google.com)
 - [Deploy](https://google.com)

### Правила оформления Pull Requests
  1. Ветка создается с названием `MUSIC-###`, где ### - номер задачи.
  2. Название Pull Request'а соответствует названию задачи: `MUSIC-###: description`,
     где description - название задачи (что вы реализовали в этом Pull Request'е).
  3. Для того, чтобы залить изменения в ветку main нужен апрув от [Тимофея](https://t.me/TimofeyChichikin)
