package env

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func String(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func Duration(key string, fallback time.Duration) (time.Duration, error) {
	raw := String(key, "")
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
