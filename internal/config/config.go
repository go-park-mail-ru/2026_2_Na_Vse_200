package config

import (
	"fmt"
	"strconv"
	"time"

	"2026_2_Na_Vse_200/internal/env"
)

// Config содержит настройки приложения.
type Config struct {
	LogLevel        string
	LogFormat       string
	Addr            string
	AllowedOrigin   string // Пустое значение означает отсутствие отдельного origin для фронтенда.
	AllowedMethods  string
	AllowedHeaders  string
	CookieSecure    bool
	SessionTTL      time.Duration
	ShutdownTimeout time.Duration
	PostgreSQLDSN   string
	MediaRoot       string
	AuthTarget      string
	MetricsAddr     string
}

// Load читает настройки из окружения и возвращает ошибку при неверном формате значения.
func Load() (cfg Config, err error) {
	cfg = Config{
		LogLevel:       env.String("LOG_LEVEL", "info"),
		LogFormat:      env.String("LOG_FORMAT", "json"),
		Addr:           env.String("APP_ADDR", ":8080"),
		AllowedOrigin:  env.String("APP_ALLOWED_ORIGIN", ""),
		AllowedMethods: env.String("APP_ALLOWED_METHODS", "GET, POST, PUT, DELETE, OPTIONS"),
		AllowedHeaders: env.String("APP_ALLOWED_HEADERS", "Content-Type, X-CSRF-Token"),
		PostgreSQLDSN:  env.String("POSTGRES_DSN", ""),
		MediaRoot:      env.String("MEDIA_ROOT", "data/media"),
		AuthTarget:     env.String("AUTH_GRPC_TARGET", "127.0.0.1:50051"),
		MetricsAddr:    env.String("APP_METRICS_ADDR", ""),
	}

	cfg.CookieSecure, err = getBool("APP_COOKIE_SECURE", false)
	if err != nil {
		return Config{}, err
	}

	cfg.SessionTTL, err = env.Duration("APP_SESSION_TTL", 24*time.Hour)
	if err != nil {
		return Config{}, err
	}

	cfg.ShutdownTimeout, err = env.Duration("APP_SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func getBool(key string, fallback bool) (bool, error) {
	raw := env.String(key, "")
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("разбор %s: %w", key, err)
	}
	return value, nil
}
