// Package config читает настройки сервиса из переменных окружения.
//
// Значения по умолчанию рассчитаны на локальный запуск: достаточно `go run ./cmd/server`,
// ничего настраивать не нужно. На стенде переменные задаются снаружи.
// Секреты в репозиторий не попадают: .env в .gitignore.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config — настройки, собранные при старте. Дальше по коду передаётся значением:
// после Load она не меняется.
type Config struct {
	// Addr — адрес прослушивания, например ":8080".
	Addr string

	// AllowedOrigins — точные адреса фронтенда для CORS. Пустой список означает,
	// что фронт и API на одном origin и CORS не нужен.
	AllowedOrigins []string

	// CookieSecure — флаг Secure у сессионной cookie. Локально false: по http://
	// браузер cookie с Secure не примет.
	CookieSecure bool

	// SessionTTL — срок жизни сессии и cookie.
	SessionTTL time.Duration

	// ShutdownTimeout — сколько ждём завершения текущих запросов при остановке.
	ShutdownTimeout time.Duration

	// DSN — строка подключения к PostgreSQL. Читается уже сейчас, используется
	// с BE-03, когда появится хранение аккаунтов.
	DSN string
}

// Load собирает конфигурацию из окружения.
//
// Ошибка возвращается, если значение задано, но разобрать его не удалось:
// молча подставить дефолт вместо опечатки в APP_SESSION_TTL — значит получить
// сюрприз на стенде вместо понятной ошибки при старте.
func Load() (Config, error) {
	cfg := Config{
		Addr: getEnv("APP_ADDR", ":8080"),
		DSN:  getEnv("DB_DSN", ""),
	}

	cfg.AllowedOrigins = splitOrigins(getEnv("APP_ALLOWED_ORIGINS", ""))

	secure, err := getBool("APP_COOKIE_SECURE", false)
	if err != nil {
		return Config{}, err
	}
	cfg.CookieSecure = secure

	sessionTTL, err := getDuration("APP_SESSION_TTL", 24*time.Hour)
	if err != nil {
		return Config{}, err
	}
	cfg.SessionTTL = sessionTTL

	shutdownTimeout, err := getDuration("APP_SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	cfg.ShutdownTimeout = shutdownTimeout

	return cfg, nil
}

// getEnv возвращает значение переменной или запасное, если она не задана или пуста.
func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

// getBool разбирает значение вида true/false/1/0.
func getBool(key string, fallback bool) (bool, error) {
	raw := getEnv(key, "")
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("разбор %s: %w", key, err)
	}
	return value, nil
}

// getDuration разбирает значение вида 24h, 30m, 10s.
func getDuration(key string, fallback time.Duration) (time.Duration, error) {
	raw := getEnv(key, "")
	if raw == "" {
		return fallback, nil
	}

	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("разбор %s: %w", key, err)
	}
	if value <= 0 {
		return 0, fmt.Errorf("разбор %s: длительность должна быть положительной, получено %q", key, raw)
	}
	return value, nil
}

// splitOrigins превращает список через запятую в срез, выбрасывая пустые элементы.
func splitOrigins(raw string) []string {
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		if origin := strings.TrimSpace(part); origin != "" {
			origins = append(origins, origin)
		}
	}

	if len(origins) == 0 {
		return nil
	}
	return origins
}
