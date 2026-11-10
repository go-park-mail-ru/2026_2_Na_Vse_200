# Тесты

**Покрытие — 59,4% без сгенерированных моков** (10.11.2026, без тестовой БД).
Требование РК2 — больше 60%.

Тесты лежат рядом с кодом в файлах `*_test.go`. Все команды выполняются из корня проекта.

## Запуск

```bash
go test ./...
```

Отдельный тест с подробным выводом:

```bash
go test ./internal/delivery/handlers -run TestSignupWithMocks -v
```

## Покрытие

В PowerShell:

```powershell
go test ./... '-coverpkg=./...' '-coverprofile=cover.out'
Get-Content cover.out | Where-Object { $_ -notmatch '/mocks/|/mock_[^/]*\.go:' } | Set-Content -Encoding ascii cover.nomocks.out
go tool cover '-func=cover.nomocks.out'
```

Общий процент — в строке `total`. Для просмотра по строкам в браузере:

```powershell
go tool cover '-html=cover.nomocks.out'
```

Из расчёта исключаются только сгенерированные моки. `-coverpkg=./...` учитывает вызовы между пакетами.

## Моки

Используются `go.uber.org/mock v0.6.0` и `github.com/DATA-DOG/go-sqlmock v1.5.2`.

- `internal/delivery/handlers/auth_gomock_test.go` — регистрация и отказ при неверном пароле.
- `internal/delivery/handlers/home_gomock_test.go` — ответ главной и ошибки репозитория.
- `internal/repository/postgres/user_repo_sqlmock_test.go` — SQL-запросы пользователей без подключения к БД. Тестовый адаптер связывает вызовы pgx с sqlmock.

Генерация моков после изменения интерфейсов:

```bash
go generate ./internal/usecase/ports
```

Директивы находятся в `internal/usecase/ports/repository.go`, результат — в `internal/usecase/ports/mocks/`. Сгенерированные файлы вручную не редактируются.

## Тесты с PostgreSQL

Нужна отдельная тестовая база и пользователь с правами на создание и удаление схем. Тесты создают временные схемы и удаляют их после выполнения. Рабочую базу приложения не используйте.

Пример для PowerShell после создания базы `music_test`:

```powershell
$env:TEST_POSTGRES_DSN = 'postgres://postgres:YOUR_PASSWORD@localhost:5432/music_test?sslmode=disable'
go test ./internal/repository/postgres ./internal/migration ./db/migrations -count=1 -v
```

Без `TEST_POSTGRES_DSN` эти проверки пропускаются. Переменная `POSTGRES_DSN` приложения для тестов не используется.
