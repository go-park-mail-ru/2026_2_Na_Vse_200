# Разработка

Быстрый запуск — в [основном README](../README.md), тесты и покрытие — в [tests/README.md](../tests/README.md).

## Структура

| Каталог | Назначение |
|---|---|
| `cmd/app` | Точка входа, команды `server` и `migrate` |
| `internal/app` | Сборка зависимостей и запуск сервера |
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

Запрос проходит через обработчик, usecase и репозиторий. Usecase работает через интерфейсы и не зависит от HTTP или драйвера БД. Зависимости собираются в `internal/app`.

## Запуск без Docker

Нужны Go и PostgreSQL. Задайте `POSTGRES_DSN` в окружении, затем выполните из корня проекта:

```bash
go run ./cmd/app migrate up
go run ./cmd/app server
```

Приложение само не читает `.env`. В Docker Compose этот файл используется для подстановки настроек в `compose.yaml`.

## Конфигурация

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `POSTGRES_DSN` | обязательно | Подключение к PostgreSQL |
| `APP_ADDR` | `:8080` | Адрес HTTP-сервера |
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
docker compose logs -f server
```

Порты меняются через `DOCKER_HTTP_PORT` и `DOCKER_POSTGRES_PORT`. PostgreSQL хранится в именованном volume, медиа — в каталоге `media/`, подключённом к контейнеру. Подробности — в [инструкции БД](../db/README.md).

## Проверки перед PR

```bash
gofmt -l .
go build ./...
go vet ./...
go test ./...
```
