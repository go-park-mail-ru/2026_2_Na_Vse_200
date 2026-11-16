# Логирование на zerolog в Music API

## 1. Что изменилось на MUSIC-54

В последней правке middleware получил два поля. Само создание логгера запроса, передача его через `context.Context` и трассировка PostgreSQL уже присутствовали в рабочем коде.

| Поле | Где появляется | Зачем смотреть |
| --- | --- | --- |
| `caller` | В событиях, записанных через логгер HTTP-запроса | Найти файл и строку вызова логирования. У access log это строка middleware, у события обработчика — строка обработчика |
| `route` | В итоговой записи HTTP-запроса | Группировать обращения к одному маршруту, например `GET /api/v1/tracks/{id}/audio`, независимо от значения ID |

`url` по-прежнему хранит фактический путь, например `/api/v1/tracks/42/audio`. `route` берётся из `r.Pattern` после обработки запроса. Оно может быть пустым, например если middleware вернул ответ до обращения к маршрутизатору.

[Код WithLogging](internal/delivery/middleware/logging.go) · [Регрессионный тест](internal/delivery/middleware/logging_regression_test.go)

## 2. Как и где создаётся логгер

1. При команде `app server` функция `run` читает конфигурацию приложения.
2. `logger.New(os.Stdout, cfg.LogLevel, cfg.LogFormat)` создаёт базовый `zerolog.Logger`. У записей есть поле `service=music-api` и время события.
3. `app.Run` кладёт базовый логгер в контекст приложения для инфраструктурных операций и передаёт его в сборку HTTP-цепочки. Этот экземпляр используется также для сообщений о запуске, остановке сервера и ошибках очистки сессий.
4. У `main` есть отдельный стартовый логгер для ошибок команды, включая ошибки конфигурации. Он пишет в `stderr`. Обычные события сервера идут в `stdout`. Docker собирает оба потока.

| Переменная | Значение по умолчанию | Как использовать |
| --- | --- | --- |
| `LOG_LEVEL` | `info` | `debug` показывает диагностические события сценариев и запросов PostgreSQL. `info` оставляет обычные события, предупреждения и ошибки |
| `LOG_FORMAT` | `json` | `json` выдаёт JSON на каждое событие. `console` печатает текст для просмотра человеком |

[Точка входа](cmd/app/main.go) · [Конфигурация](internal/config/config.go) · [Конструктор логгера](internal/logger/logger.go) · [Запуск и остановка HTTP](internal/app/http.go)

## 3. Что происходит с контекстом HTTP-запроса

`context.Context` — интерфейс Go для передачи сигнала отмены, дедлайна и данных текущей операции. `ctx` — обычное имя переменной этого типа. Например, когда клиент разрывает соединение, драйвер БД может получить сигнал отмены через переданный ему контекст.

Первый контекст создаёт HTTP-сервер. Middleware добавляют данные в контекст, получая производные контексты. У каждого запроса своя цепочка, поэтому у параллельных запросов разные логгеры и идентификаторы.

```text
Клиент → WithRequestID → WithLogging → WithRecover
  → WithCORS → WithCSRF → WithJSONErrors → ServeMux
  → обработчик → use case → репозиторий → pgx → PostgreSQL
  ← HTTP-ответ, затем access log
```

1. `WithRequestID` читает входной `X-Request-ID`. Если там корректный UUID, использует его, иначе генерирует свой. Сохраняет ID в контексте и ставит заголовок ответа.
2. `WithLogging` создаёт дочерний логгер из базового с `request_id` и `Caller()`. Вызов `requestLogger.WithContext(r.Context())` сохраняет его в производный контекст. `r.WithContext(ctx)` передаёт этот контекст дальше вместе с запросом.
3. На защищённых маршрутах `WithAuth` после проверки сессии создаёт ещё один дочерний логгер с `user_id` для обработчика и его вызовов. Внешний access log формируется исходным логгером запроса, поэтому общим полем для поиска остаётся `request_id`.
4. В обработчике читают `r.Context()`, в use case и репозитории передают его параметром `ctx`. `zerolog.Ctx(ctx)` извлекает ближайший логгер из этой цепочки.

Из вложенного примера в проекте используется тот же приём с дочерним логгером. Базовый экземпляр передаётся явно в `WithLogging`, поэтому тест может подставить логгер, который пишет в буфер. HTTP-маршрут берётся из стандартного `ServeMux`. `metadata.NewOutgoingContext` из примера предназначен для исходящих gRPC-вызовов, а `GetBodyCopy` читает и восстанавливает тело запроса. Для текущей HTTP-цепочки достаточно уже доступных метаданных запроса.

[Request ID](internal/delivery/middleware/request_id.go) · [HTTP-цепочка](internal/app/http.go) · [Авторизация](internal/delivery/middleware/auth.go) · [Go context](https://pkg.go.dev/context) · [zerolog.WithContext](https://github.com/rs/zerolog/blob/v1.34.0/ctx.go)

## 4. Один запрос от начала до конца

Возьмём `GET /api/v1/home`. Он удобен для проверки, потому что проходит через каталог к базе без авторизации.

| Шаг | Что делает код | Что искать в логах |
| --- | --- | --- |
| HTTP | `ServeMux` вызывает `Home`, а обработчик передаёт `r.Context()` в `h.catalog.Home` | Этот HTTP-запрос завершается итоговой записью с `route=GET /api/v1/home` |
| Use case | `Usecase.Home` вызывает `logger.UsecaseStarted(ctx, "catalog.home")`, затем получает треки, исполнителей и альбомы | Событие `debug` с `layer=usecase`, `operation=catalog.home` и `phase=start`. Успешные чтения также записывают `count` на уровне `info` |
| Репозиторий | Например, `HomeTracks` добавляет в производный контекст имя `catalog.home_tracks`, а `readTracks` вызывает `pool.Query(ctx, ...)` | `phase=start` и `phase=finish` с `layer=repository` и `operation=catalog.home_tracks` |
| PostgreSQL | При создании пула приложение подключает `QueryTracer`. pgx вызывает его при выполнении запроса | При завершении видны `duration_ms`, `outcome`, при успехе — `rows_affected`. Трассировщик записывает тип выражения, например `SELECT`, и исключает текст SQL и параметры из своих событий |
| HTTP-ответ | Обработчик пишет JSON, затем `WithLogging` записывает результат | `status=200`, `response_bytes`, `duration_ms`, `caller`, `route` и тот же `request_id` |

За статусом и количеством байт следит `statusWriter`. `content_length` относится к входящему телу, `response_bytes` — к байтам тела ответа, записанным через обёртку. Время access log включает обработку запроса и запись ответа. Время отдельного SQL-события относится к вызову pgx. Это разные замеры.

[Обработчик главной](internal/delivery/handlers/home/handler.go) · [Сценарий каталога](internal/usecase/catalog/usecase.go) · [Репозиторий каталога](internal/repository/postgres/catalog_repo.go) · [QueryTracer](internal/repository/postgres/query_tracer.go) · [statusWriter](internal/delivery/middleware/chain.go)

## 5. Как добавить своё событие

В функции, которая уже принимает `ctx`, можно записать событие так:

```go
zerolog.Ctx(ctx).Error().
    Str("operation", "profile.get").
    Err(err).
    Msg("не удалось прочитать профиль")
```

- `Error()` — уровень события.
- `Str("operation", "profile.get")` — стабильное имя для фильтрации.
- `Err(err)` — причина, записанная в отдельное поле `error`.
- `Msg(...)` — вызов, который отправляет событие. Сообщение описывает действие, а поле `error` содержит технические детали.
- `request_id`, `caller` и после авторизации `user_id` берутся из настроенного логгера запроса. Их не нужно передавать в каждую функцию отдельными параметрами.

Для входа в use case в проекте есть `logger.UsecaseStarted(ctx, "profile.get")`. Для SQL достаточно передать контекст в методы pgx и при необходимости задать имя операции через `withDBOperation` внутри пакета репозитория. При диагностике один `caller` показывает строку записи события, а `operation` помогает понять, какой сценарий или SQL-вызов за этой строкой стоит.

Уровень итоговой HTTP-записи выбирается по статусу: `4xx` даёт `warn`, `5xx` — `error`, обычные ответы — `info`. При ошибке БД обработчик может отправить клиенту стандартный JSON с `500`, а причину записать в серверный лог. `WithRecover` использует тот же request ID для записи паники и стека.

При добавлении полей используй технические данные, которые нужны для диагностики. Пароли, токены, заголовок `Authorization` и содержимое сессионных cookie в события добавлять не нужно. Штатный access log использует `r.URL.Path`, поэтому строка запроса из URL в это поле не попадает.

[Пример чтения профиля](internal/usecase/profile/usecase.go) · [Перехват паники](internal/delivery/middleware/recover.go) · [Документация zerolog v1.34.0](https://github.com/rs/zerolog/tree/v1.34.0)

## 6. Как запустить и найти логи

Если `docker info` выводит секцию `Client`, а под `Server` сообщает, что `dockerDesktopLinuxEngine` отсутствует, CLI установлен, но Linux-движок Docker пока недоступен. Открой Docker Desktop или запусти его из PowerShell. Проверяй запуск по версии сервера:

```powershell
docker desktop start
docker info --format '{{.ServerVersion}}'
```

Запускай Compose после того, как вторая команда вернёт версию. Сервис `db` запускает PostgreSQL, `migrate` применяет миграции, `server` запускает API. После успешного запуска одноразовый `migrate` может быть в состоянии `Exited (0)`.

```powershell
$env:LOG_LEVEL = 'debug'
$env:LOG_FORMAT = 'json'
docker compose up --build -d
docker compose ps -a
```

Значения, заданные через `$env:`, действуют в текущем PowerShell. Для повторных запусков их можно установить в локальном `.env`. Готовый контейнер получает настройки при создании, поэтому после смены уровня нужен новый `docker compose up -d server`, чтобы Compose применил обновлённую конфигурацию.

В следующем примере PowerShell получит ID из заголовка ответа и найдёт записи по нему. Используй внешний порт из `docker compose ps`, если он отличается от `8080`.

```powershell
$response = Invoke-WebRequest -Uri http://127.0.0.1:8080/api/v1/home
$response.StatusCode
$id = $response.Headers['X-Request-ID'] | Select-Object -First 1
$id
docker compose logs --no-color --since 2m server | Select-String -SimpleMatch $id
```

В выводе найди вход в `catalog.home`, пары событий SQL и итоговую запись запроса. Для просмотра в реальном времени используй `docker compose logs -f server`. Нажатие Ctrl+C при просмотре остановит вывод в этом терминале, а контейнер продолжит работать.

При проверке в Docker `GET /health` и `GET /api/v1/home` вернули `200`. Для запроса главной страницы по одному ID нашлись вход в `catalog.home`, три пары событий PostgreSQL для треков, исполнителей и альбомов и итоговая запись с `route=GET /api/v1/home`. В этих событиях `caller` указывал соответственно на функцию логирования use case, трассировщик PostgreSQL и HTTP-middleware.

[Настройки Compose](compose.yaml) · [Запуск Docker Desktop](https://docs.docker.com/reference/cli/docker/desktop/start/)
