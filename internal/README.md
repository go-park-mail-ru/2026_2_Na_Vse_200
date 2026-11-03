# Разработка

Техническая инструкция по запуску и настройке сервиса. Контракт API — в [docs/api.md](../docs/api.md).
Схема, миграции и ограничения БД — в [db/README.md](../db/README.md).

## Запуск через Docker Compose

Нужен Docker Desktop в режиме Linux containers либо Docker Engine с Compose. Из корня проекта:

```powershell
docker info
docker compose up --build -d
docker compose ps -a
```

Дождитесь готовности Docker Engine. Ошибка подключения к named pipe означает, что Engine недоступен: запустите Docker Desktop и повторите `docker info`.

Откройте [http://localhost:8080/health](http://localhost:8080/health). Compose запускает PostgreSQL, ждёт его готовности, применяет миграции и затем запускает API. Сервис `migrate` после успеха завершается с кодом `0`; `db` и `server` продолжают работать.

Для первого локального запуска `.env` создавать не обязательно. Существующий `.env` не перезаписывайте: контейнеры получают только явно выбранные настройки из `compose.yaml`. Пароль разработки по умолчанию — `music_local_dev`; его настройка и подключение SQLTools описаны в [инструкции БД](../db/README.md#вся-система-в-docker-compose).

API доступен на `localhost:8080`, отдельная база Docker — на `localhost:5433`. Уже установленная база на порту `5432` остаётся отдельной. Внутри Compose Go подключается к сервису `db:5432`; локальный `POSTGRES_DSN` из `.env` для этого не используется. Настройки `DOCKER_HTTP_PORT` и `DOCKER_POSTGRES_PORT` меняют внешние порты.

```powershell
docker compose logs migrate
docker compose logs -f server
docker compose run --rm migrate version
docker compose exec db psql -U music -d music
docker compose down
```

`down` останавливает контейнеры и сохраняет базу в именованном volume. После изменения кода или SQL-миграций повторите `docker compose up --build -d`: команда пересобирает образ и запускает миграции. Обычный `restart` этого не делает.

## Команды приложения

В проекте одна точка входа cmd/app/main.go. Собранная программа app принимает
команду server для запуска HTTP-сервера или migrate для работы с миграциями.

```text
app server
app migrate up
app migrate version
app migrate down 1
```

Для локального запуска через Go используются go run ./cmd/app server
и go run ./cmd/app migrate up. Настройки должны быть заданы в окружении;
Go-программа сама .env не загружает. В Docker это делает конфигурация Compose.

Dockerfile собирает один бинарный файл /app/app. Сервис server передаёт
ему команду server, сервис migrate — migrate up. Сохраняется короткий вызов
docker compose run --rm migrate version: entrypoint этого сервиса уже содержит
команду migrate.

## Конфигурация

Настройки читаются из переменных окружения. `POSTGRES_DSN` обязателен; остальные значения по умолчанию рассчитаны на локальный запуск.

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `APP_ADDR` | `:8080` | адрес, который слушает сервер |
| `APP_ALLOWED_ORIGIN` | пусто | адрес фронтенда для CORS; пусто — фронт и API на одном origin |
| `APP_ALLOWED_METHODS` | `GET, POST, OPTIONS` | методы, разрешённые кросс-доменному запросу |
| `APP_ALLOWED_HEADERS` | `Content-Type` | заголовки, разрешённые кросс-доменному запросу |
| `APP_COOKIE_SECURE` | `false` | флаг `Secure` у сессионной cookie; на стенде с HTTPS — `true` |
| `APP_SESSION_TTL` | `24h` | срок жизни сессии |
| `APP_SHUTDOWN_TIMEOUT` | `10s` | сколько ждём завершения запросов при остановке |
| `POSTGRES_DSN` | пусто, обязателен | строка подключения к PostgreSQL |

Для локального HTTP с фронтендом на другом порту того же хоста задайте
`APP_ALLOWED_ORIGIN=http://localhost:3000` и оставьте `APP_COOKIE_SECURE=false`.
Cookie будет иметь `SameSite=Lax`. При заданном origin и `APP_COOKIE_SECURE=true`
сервер выбирает `SameSite=None`; этот вариант предназначен для HTTPS.

### Секреты

Шаблон `.env.example` содержит только пример настроек без секретов. Свои значения держите локально в `.env` (он в `.gitignore`), на стенде — в переменных окружения сервера, а для сборки и деплоя их место в GitHub Secrets. Токены внешних сервисов для запуска БД и сервера не нужны.

### Настройки Docker Compose

Compose читает .env из корня проекта и подставляет значения в compose.yaml.
Для первого запуска достаточно значений по умолчанию. Существующий .env
сохраняйте; доступные настройки перечислены в .env.example.

Сервер и миграции внутри контейнеров подключаются к db:5432.
POSTGRES_DSN и PGPASSWORD для них задаёт compose.yaml.
Внешние порты меняются через DOCKER_HTTP_PORT и DOCKER_POSTGRES_PORT.

Для другого файла настроек используйте:

```powershell
docker compose --env-file .env.stand up --build -d
```

## Структура

```
Dockerfile                  сборка Linux-образа с единой программой app
compose.yaml                PostgreSQL, миграции, API и постоянный volume
cmd/app/                    точка входа: команды server и migrate
internal/app/               подключение к БД, сборка зависимостей и запуск сервера
internal/migration/         разбор команды, подключение к БД и транзакции миграций
db/migrations/             SQL-схема и история её изменений
db/normalized/             модель данных и ER-диаграмма
internal/auth/              хеширование и проверка паролей
internal/config/            настройки из переменных окружения
internal/delivery/handlers/routes.go сборка маршрутов
internal/delivery/handlers/auth/     авторизация, cookie, запросы и ответы
internal/delivery/handlers/home/     главная страница, ответы и преобразования
internal/delivery/handlers/health/   проверка доступности сервиса
internal/delivery/handlers/repositories_test.go локальные неэкспортируемые подмены для unit-тестов
internal/delivery/middleware/        обвязка: request_id, recover, логирование, CORS, JSON-ошибки
internal/models/            доменные сущности, коды ошибок и тексты для пользователя
internal/usecase/auth/      регистрация, вход, текущий пользователь, выход
internal/usecase/catalog/   сбор данных главной страницы
internal/usecase/ports/     контракты хранилищ и хеширования для приложения
internal/models/errors.go  общие ошибки предметной области
internal/repository/postgres/ аккаунты, сессии и каталог в PostgreSQL
internal/delivery/response/               запись JSON-ответов
internal/validation/             проверки данных от клиента
```

## Границы архитектуры

Запрос проходит через HTTP-обработчик, usecase и реализацию репозитория.
Go-зависимости направлены к внутренним слоям:

- `models` описывает сущности и общие ошибки и не импортирует инфраструктуру;
- `usecase` выполняет сценарии через интерфейсы `usecase/ports`, не знает об HTTP,
  cookie, SQL и драйвере PostgreSQL;
- `repository/postgres` реализует контракты приложения и переводит ошибки БД
  в общие ошибки моделей;
- `handlers` зависит от интерфейсов сценариев, читает JSON, устанавливает cookie
  и переводит результаты и ошибки в HTTP-ответы;
- `internal/app/app.go` создаёт PostgreSQL-репозитории, bcrypt-адаптер и usecase,
  затем передаёт сценарии обработчикам.

Время, генератор идентификаторов и TTL передаются в usecase авторизации извне.
Валидация и нормализация выполняются внутри сценария, поэтому действуют и при
вызове без HTTP. Структуры запросов и ответов JSON остаются в `handlers`.
Новые сценарии используют собственные входные и выходные Go-структуры.

В ports находятся контракты пользователей, сессий, каталога и хеширования,
а также минимальные контракты профиля, метаданных файлов, исполнителей и треков.
Последние реализованы в PostgreSQL, но пока не подключены к usecase и HTTP.
Загрузка аватара через multipart/form-data и выдача аудио с HTTP Range требуют
отдельных обработчиков и файлового слоя. Схема БД сохранена.
Сессии хранятся в `auth_session` и сохраняются после перезапуска backend. Истёкшая сессия отклоняется при чтении по времени PostgreSQL. Очистка одним `DELETE` запускается в фоне при старте параллельно запуску HTTP-сервера и затем раз в час, с ограничением времени выполнения; запуск HTTP-сервера её завершения не ждёт. Redis не реализован; обязательного переноса сессий туда нет.

Для аудио и изображений выбран диск сервера. В `media_file` хранится метаинформация и `storage_key` — относительный ключ внутри одного корня будущего файлового слоя, а не абсолютный путь или URL. Условное имя корня — `MEDIA_ROOT`; действующая конфигурация и volume для него ещё не добавлены. При реализации каталог хранения нужно размещать вне репозитория и сохранять независимо от пересоздания контейнера. Загрузка, скачивание и построение публичных URL пока не реализованы, поэтому адреса изображений в API передаются как `null`. Выдача аудио с HTTP `Range` планируется вместе с файловым слоем. Подробнее — [модель хранения медиа](../db/README.md#медиафайлы-на-диске-сервера).

## Проверки перед PR

```
go build ./...
go vet ./...
gofmt -l .
```

Команды тестов, измерение покрытия и описание проверок — в [инструкции тестирования](../tests/README.md).
