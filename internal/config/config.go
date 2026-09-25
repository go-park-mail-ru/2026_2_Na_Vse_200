// Package config читает настройки сервиса из переменных окружения.
// Значений по умолчанию достаточно для локального запуска.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config — настройки, собранные при старте.
type Config struct {
	// Addr — адрес прослушивания, например ":8080".
	Addr string

	// AllowedOrigin — адрес фронтенда для CORS.
	// Пустая строка означает общий с фронтендом origin: CORS не нужен.
	AllowedOrigin string

	// CookieSecure — флаг Secure у сессионной cookie. По http:// должен быть false.
	CookieSecure bool

	// SessionTTL — срок жизни сессии и cookie.
	SessionTTL time.Duration

	// ShutdownTimeout — сколько ждём завершения запросов при остановке.
	ShutdownTimeout time.Duration

	// PostgreSQLDSN — строка подключения к PostgreSQL.
	PostgreSQLDSN string
}

// Load собирает конфигурацию из окружения. Заданное, но неразбираемое значение —
// ошибка, а не повод взять значение по умолчанию.
func Load() (Config, error) {
	cfg := Config{
		Addr:          getEnv("APP_ADDR", ":8080"),
		AllowedOrigin: getEnv("APP_ALLOWED_ORIGIN", ""),
		PostgreSQLDSN: getEnv("POSTGRES_DSN", ""),
	}

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

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

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

// getDuration разбирает значения вида 24h, 30m, 10s.
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
