# Разработка

Техническая инструкция по запуску и настройке сервиса. Контракт API — в [docs/api.md](../docs/api.md).
Схема, миграции и ограничения БД — в [db/README.md](../db/README.md).

## Запуск через Docker Compose

Нужен Docker Desktop в режиме Linux containers либо Docker Engine с Compose. Из корня проекта:

```powershell
docker compose up --build -d
docker compose ps -a
```

Откройте [http://localhost:8080/health](http://localhost:8080/health). Compose запускает PostgreSQL, ждёт его готовности, применяет миграции и затем запускает API. Сервис `migrate` после успеха завершается с кодом `0`; `db` и `server` продолжают работать.

Для первого локального запуска `.env` создавать не обязательно. Существующий `.env` не перезаписывайте: контейнеры получают только явно выбранные настройки из `compose.yaml`. Пароль разработки по умолчанию — `music_local_dev`; его настройка и подключение SQLTools описаны в [инструкции БД](../db/README.md#запуск-через-docker-compose).

API доступен на `localhost:8080`, отдельная база Docker — на `localhost:5433`. Уже установленная база на порту `5432` остаётся отдельной. Внутри Compose Go подключается к сервису `db:5432`; локальный `POSTGRES_DSN` из `.env` для этого не используется. Настройки `DOCKER_HTTP_PORT` и `DOCKER_POSTGRES_PORT` меняют внешние порты.

```powershell
docker compose logs migrate
docker compose logs -f server
docker compose run --rm migrate version
docker compose exec db psql -U music -d music
docker compose down
```

`down` останавливает контейнеры и сохраняет базу в именованном volume. После изменения кода или SQL-миграций повторите `docker compose up --build -d`: команда пересобирает образ и запускает миграции. Обычный `restart` этого не делает.

## Запуск без Docker

Нужны Go версии из `go.mod` и PostgreSQL. Перед первым запуском создайте пустую базу и примените SQL-миграции по [инструкции БД](../db/README.md). Аккаунты и музыкальный каталог читаются из PostgreSQL; без `POSTGRES_DSN` сервер не запустится.

Если `.env` ещё нет, создайте его по `.env.example`. Если файл уже существует, добавьте настройку базы в него, не перезаписывая остальные значения.

PowerShell из корня проекта:

```powershell
.\scripts\run.ps1 go run ./cmd/migrate up
.\scripts\run.ps1
```

Bash/Git Bash:

```bash
./scripts/run.sh go run ./cmd/migrate up
./scripts/run.sh
```

Если переменные уже заданы в окружении, сервер можно запустить напрямую:

```
go run ./cmd/server
```

Миграции при старте сервера автоматически не выполняются. Не применяйте начальные миграции вслепую к базе, таблицы которой ранее создавались вручную; для разработки используйте новую пустую базу либо отдельно согласованный baseline.

Сервер поднимается на http://localhost:8080. Проверка:

```
curl http://localhost:8080/health      # {"status":"ok"}
```

Остановка — Ctrl+C: сервер дожидается текущих запросов и только потом завершается.

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

### Скрипт запуска

`scripts/run.ps1` для PowerShell и `scripts/run.sh` для Bash подхватывают локальную `.env` и запускают указанную команду; без аргументов запускают сервер.

```powershell
.\scripts\run.ps1
.\scripts\run.ps1 go run ./cmd/migrate version
.\scripts\run.ps1 go run ./cmd/migrate down 1
.\scripts\run.ps1 go test ./...
```

```bash
./scripts/run.sh
./scripts/run.sh go run ./cmd/migrate version
./scripts/run.sh go test ./...
ENV_FILE=.env.stand ./scripts/run.sh
```

Если `.env` отсутствует, обязательный `POSTGRES_DSN` нужно передать через окружение. На Windows Bash-скрипт запускается из Git Bash.

## Структура

```
Dockerfile                  сборка Linux-образа сервера и команды миграций
compose.yaml                PostgreSQL, миграции, API и постоянный volume
cmd/server/                 точка входа: конфигурация, логгер, обвязка, запуск
cmd/migrate/                команды применения и отката SQL-миграций через pgx
db/migrations/             SQL-схема и история её изменений
db/normalized/             модель данных и ER-диаграмма
internal/apimessage/        коды ошибок и тексты для пользователя
internal/auth/              хеширование и проверка паролей
internal/config/            настройки из переменных окружения
internal/handlers/          HTTP-обработчики
internal/middleware/        обвязка: request_id, recover, логирование, CORS, JSON-ошибки
internal/models/            доменные сущности
internal/repository/        интерфейсы доступа к данным и ошибки хранилища
internal/repository/postgres/ аккаунты и каталог в PostgreSQL
internal/repository/memory/  сессии в памяти; также реализации для проверок
pkg/response/               запись JSON-ответов
pkg/validation/             проверки данных от клиента
```

Сессии пока живут в памяти процесса и теряются после его перезапуска; Redis остаётся запланированным хранилищем. Таблицы `media_file` описывают расположение аудио и изображений, однако S3 и построение публичных URL ещё не подключены. Пустые адреса изображений в ответе передаются как `null`.

## Проверки перед PR

```
go build ./...
go vet ./...
gofmt -l .
```

Команды тестов, измерение покрытия и описание проверок — в [инструкции тестирования](../tests/README.md).