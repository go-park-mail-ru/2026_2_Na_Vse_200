package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	// Пустое окружение: сервис должен подниматься локально без настройки.
	t.Setenv("APP_ADDR", "")
	t.Setenv("APP_ALLOWED_ORIGINS", "")
	t.Setenv("APP_COOKIE_SECURE", "")
	t.Setenv("APP_SESSION_TTL", "")
	t.Setenv("APP_SHUTDOWN_TIMEOUT", "")
	t.Setenv("DB_DSN", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: неожиданная ошибка: %v", err)
	}

	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q, ожидался %q", cfg.Addr, ":8080")
	}
	if cfg.SessionTTL != 24*time.Hour {
		t.Errorf("SessionTTL = %v, ожидалось %v", cfg.SessionTTL, 24*time.Hour)
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Errorf("ShutdownTimeout = %v, ожидалось %v", cfg.ShutdownTimeout, 10*time.Second)
	}
	if cfg.CookieSecure {
		t.Error("CookieSecure = true, локально по http:// браузер такую cookie не примет")
	}
	if len(cfg.AllowedOrigins) != 0 {
		t.Errorf("AllowedOrigins = %v, ожидался пустой список (CORS выключен)", cfg.AllowedOrigins)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("APP_ADDR", ":9000")
	t.Setenv("APP_ALLOWED_ORIGINS", "http://localhost:3000, https://navse200.ru ,")
	t.Setenv("APP_COOKIE_SECURE", "true")
	t.Setenv("APP_SESSION_TTL", "1h30m")
	t.Setenv("APP_SHUTDOWN_TIMEOUT", "5s")
	t.Setenv("DB_DSN", "postgres://user:pass@localhost:5432/navse200")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: неожиданная ошибка: %v", err)
	}

	if cfg.Addr != ":9000" {
		t.Errorf("Addr = %q, ожидался %q", cfg.Addr, ":9000")
	}
	if !cfg.CookieSecure {
		t.Error("CookieSecure = false, ожидалось true")
	}
	if cfg.SessionTTL != 90*time.Minute {
		t.Errorf("SessionTTL = %v, ожидалось %v", cfg.SessionTTL, 90*time.Minute)
	}
	if cfg.ShutdownTimeout != 5*time.Second {
		t.Errorf("ShutdownTimeout = %v, ожидалось %v", cfg.ShutdownTimeout, 5*time.Second)
	}
	if cfg.DSN == "" {
		t.Error("DSN пуст, ожидалась строка подключения из окружения")
	}

	// Пробелы обрезаются, пустой элемент после последней запятой отбрасывается.
	want := []string{"http://localhost:3000", "https://navse200.ru"}
	if len(cfg.AllowedOrigins) != len(want) {
		t.Fatalf("AllowedOrigins = %v, ожидалось %v", cfg.AllowedOrigins, want)
	}
	for i, origin := range want {
		if cfg.AllowedOrigins[i] != origin {
			t.Errorf("AllowedOrigins[%d] = %q, ожидался %q", i, cfg.AllowedOrigins[i], origin)
		}
	}
}

func TestLoadInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "срок сессии не разбирается", key: "APP_SESSION_TTL", value: "сутки"},
		{name: "срок сессии отрицательный", key: "APP_SESSION_TTL", value: "-1h"},
		{name: "таймаут остановки не разбирается", key: "APP_SHUTDOWN_TIMEOUT", value: "быстро"},
		{name: "флаг cookie не разбирается", key: "APP_COOKIE_SECURE", value: "ага"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.key, tt.value)

			// Опечатка в настройке должна ронять старт с понятной ошибкой,
			// а не тихо подставлять значение по умолчанию.
			if _, err := Load(); err == nil {
				t.Errorf("Load с %s=%q: ошибки нет, а ожидалась", tt.key, tt.value)
			}
		})
	}
}
