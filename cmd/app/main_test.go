package main

import (
	"strings"
	"testing"
)

// TestCommands проверяет справку и ошибки аргументов команд.
func TestCommands(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"нет команды", nil, usage},
		{"неизвестная команда", []string{"unknown"}, "unknown command"},
		{"аргументы сервера", []string{"server", "extra"}, "server does not accept arguments"},
		{"аргументы справки", []string{"help", "extra"}, usage},
		{"справка", []string{"--help"}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := execute(tc.args)
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ошибка = %v, ожидалась %q", err, tc.want)
			}
		})
	}
}

func TestServerInvalidConfig(t *testing.T) {
	t.Setenv("APP_SESSION_TTL", "invalid")
	err := execute([]string{"server"})
	if err == nil || !strings.Contains(err.Error(), "APP_SESSION_TTL") {
		t.Fatalf("ожидалась ошибка настройки TTL, получено %v", err)
	}
}
