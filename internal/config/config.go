package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config содержит настройки приложения.
type Config struct {
	Addr            string
	AllowedOrigin   string // Пустое значение означает отсутствие отдельного origin для фронтенда.
	AllowedMethods  string
	AllowedHeaders  string
	CookieSecure    bool
	SessionTTL      time.Duration
	ShutdownTimeout time.Duration
	PostgreSQLDSN   string
	MediaRoot       string
}

// Load читает настройки из окружения и возвращает ошибку при неверном формате значения.
func Load() (cfg Config, err error) {
	cfg = Config{
		Addr:           getEnv("APP_ADDR", ":8080"),
		AllowedOrigin:  getEnv("APP_ALLOWED_ORIGIN", ""),
		AllowedMethods: getEnv("APP_ALLOWED_METHODS", "GET, POST, PUT, DELETE, OPTIONS"),
		AllowedHeaders: getEnv("APP_ALLOWED_HEADERS", "Content-Type, X-CSRF-Token"),
		PostgreSQLDSN:  getEnv("POSTGRES_DSN", ""),
		MediaRoot:      getEnv("MEDIA_ROOT", "data/media"),
	}

	cfg.CookieSecure, err = getBool("APP_COOKIE_SECURE", false)
	if err != nil {
		return Config{}, err
	}

	cfg.SessionTTL, err = getDuration("APP_SESSION_TTL", 24*time.Hour)
	if err != nil {
		return Config{}, err
	}

	cfg.ShutdownTimeout, err = getDuration("APP_SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

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

func getDuration(key string, fallback time.Duration) (time.Duration, error) {
	raw := getEnv(key, "")
	if raw == "" {
		return fallback, nil
	}

	value, err := time.ParseDuration(raw)

	switch {
	case err != nil:
		return 0, fmt.Errorf("разбор %s: %w", key, err)
	case value <= 0:
		return 0, fmt.Errorf("разбор %s: длительность должна быть положительной, получено %q", key, raw)
	default:
		return value, nil
	}
}
