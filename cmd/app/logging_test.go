package main

import (
	"context"
	"strings"
	"testing"
)

func TestRunRejectsInvalidLoggingBeforeDatabase(t *testing.T) {
	t.Setenv("LOG_LEVEL", "wrong-level")
	t.Setenv("LOG_FORMAT", "json")
	t.Setenv("POSTGRES_DSN", "")

	err := run(context.Background(), []string{"server"})
	if err == nil || !strings.Contains(err.Error(), "настройка логирования") || !strings.Contains(err.Error(), "wrong-level") {
		t.Fatalf("ожидалась ошибка уровня логирования, получено %v", err)
	}
}
