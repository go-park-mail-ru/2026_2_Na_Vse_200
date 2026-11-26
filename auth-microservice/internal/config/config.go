package config

import (
	"time"

	"2026_2_Na_Vse_200/internal/env"
)

type Config struct {
	ListenAddr      string
	MetricsAddr     string
	PostgreSQLDSN   string
	LogLevel        string
	LogFormat       string
	SessionTTL      time.Duration
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	sessionTTL, err := env.Duration("APP_SESSION_TTL", 24*time.Hour)
	if err != nil {
		return Config{}, err
	}
	shutdownTimeout, err := env.Duration("APP_SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	return Config{
		ListenAddr:      env.String("AUTH_GRPC_ADDR", "127.0.0.1:50051"),
		MetricsAddr:     env.String("AUTH_METRICS_ADDR", "127.0.0.1:9092"),
		PostgreSQLDSN:   env.String("POSTGRES_DSN", ""),
		LogLevel:        env.String("LOG_LEVEL", "info"),
		LogFormat:       env.String("LOG_FORMAT", "json"),
		SessionTTL:      sessionTTL,
		ShutdownTimeout: shutdownTimeout,
	}, nil
}
