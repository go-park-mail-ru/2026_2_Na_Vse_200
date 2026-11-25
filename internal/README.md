# Разработка

Тесты и покрытие описаны в [tests/README.md](../tests/README.md).

## Структура

| Каталог | Назначение |
|---|---|
| `cmd/app` | Точка входа, команды `server` и `migrate` |
| `cmd/auth` | Точка входа сервиса сессий и проверки доступа |
| `api/auth/v1` | Контракт Auth gRPC и сгенерированный Go-код |
| `internal/app` | Сборка зависимостей и запуск HTTP и Auth |
| `internal/delivery` | HTTP-обработчики, маршруты, middleware и JSON-ответы |
| `internal/usecase` | Бизнес-логика |
| `internal/usecase/ports` | Интерфейсы зависимостей и директивы генерации моков |
| `internal/repository/postgres` | Репозитории PostgreSQL |
| `internal/models` | Модели и общие ошибки |
| `internal/auth` | Хеширование паролей и генерация ID сессий |
| `internal/storage` | Хранение файлов на диске |
| `internal/config` | Чтение настроек из окружения |
| `internal/logger` | Настройка логирования |
| `internal/security` | CSRF и обработка текста |
| `internal/validation` | Проверка входных данных |
| `internal/migration` | Выполнение миграций |
| `db/migrations` | SQL-миграции |

HTTP-сервер вызывает Auth по gRPC для выдачи, проверки и отзыва сессий, а также для проверки доступа к профилю. Обработчики плейлистов получают идентификатор пользователя из проверенной сессии и работают с плейлистами этого пользователя.

HTTP и Auth пишут структурированные логи через zerolog. HTTP `X-Request-ID` передаётся в gRPC metadata под именем `request_id` и связывает логи обоих процессов.

## Запуск без Docker

Нужны Go и PostgreSQL. Задайте `POSTGRES_DSN` в окружении, затем выполните из корня проекта:

```bash
go run ./cmd/app migrate up
go run ./cmd/auth
```

В другом терминале с тем же `POSTGRES_DSN` и `APP_SESSION_TTL` запустите HTTP-сервер.

```bash
go run ./cmd/app server
```

Приложение само не читает `.env`. В Docker Compose этот файл используется для подстановки настроек в `compose.yaml`.

## Конфигурация

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `POSTGRES_DSN` | обязательно | Подключение к PostgreSQL |
| `APP_ADDR` | `:8080` | Адрес HTTP-сервера |
| `AUTH_GRPC_ADDR` | `127.0.0.1:50051` | Адрес gRPC-сервера Auth |
| `AUTH_GRPC_TARGET` | `127.0.0.1:50051` | Адрес Auth для HTTP-сервера |
| `APP_METRICS_ADDR` | Пусто | Адрес общих HTTP- и gRPC-метрик HTTP-сервера. Пустое значение отключает endpoint при локальном запуске. |
| `AUTH_METRICS_ADDR` | `127.0.0.1:9092` | Адрес метрик Auth |
| `APP_ALLOWED_ORIGIN` | пусто | Разрешённый origin фронтенда для CORS |
| `APP_ALLOWED_METHODS` | `GET, POST, PUT, DELETE, OPTIONS` | Методы для CORS |
| `APP_ALLOWED_HEADERS` | `Content-Type, X-CSRF-Token` | Заголовки для CORS |
| `APP_COOKIE_SECURE` | `false` | Флаг Secure у cookie; для HTTPS задаётся `true` |
| `APP_SESSION_TTL` | `24h` | Срок жизни сессии |
| `APP_SHUTDOWN_TIMEOUT` | `10s` | Время завершения запросов при остановке |
| `MEDIA_ROOT` | `data/media` | Каталог аудио и изображений |

Если фронтенд и API доступны с одного origin, `APP_ALLOWED_ORIGIN` оставьте пустым. Секреты и локальный `.env` не добавляйте в репозиторий.

## Docker Compose

После изменения кода или миграций пересоберите приложение:

```bash
docker compose up --build -d
```

Проверка версии миграций и логи:

```bash
docker compose run --rm migrate version
docker compose logs migrate
docker compose logs -f server auth
```

Порты меняются через `DOCKER_HTTP_PORT` и `DOCKER_POSTGRES_PORT`. PostgreSQL хранится в именованном volume, медиа — в каталоге `media/`, подключённом к контейнеру. Подробности — в [инструкции БД](../db/README.md).

Auth использует доверенное внутреннее соединение. В Compose HTTP-сервер отдаёт HTTP- и gRPC-метрики вместе на `server:9091/metrics`. Конфигурация мониторинга переносит этот endpoint на `server:9090/metrics`, откуда его опрашивает Prometheus. Метрики самого Auth доступны на `auth:9092/metrics`.

## Генерация контракта Auth

Из корня репозитория запустите следующие команды с `protoc` 34.1. Каталог `bin` из официального дистрибутива `protoc` и каталог Go-бинарников должны быть в `PATH`.

```text
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.0
protoc -I . --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative api/auth/v1/auth.proto
```

Сгенерированные `auth.pb.go` и `auth_grpc.pb.go` хранятся рядом с контрактом. Если `protoc` не находит стандартные импорты Google, передайте дополнительный `-I` с путём к каталогу `include` из дистрибутива.

## Проверки перед PR

```bash
gofmt -l .
go build ./...
go vet ./...
go test ./...
```
