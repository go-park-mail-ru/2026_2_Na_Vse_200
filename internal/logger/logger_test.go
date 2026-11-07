package logger_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"2026_2_Na_Vse_200/internal/logger"
)

func TestNewJSONLogger(t *testing.T) {
	for _, tt := range []struct {
		name   string
		level  string
		format string
		levels []string
	}{
		{"defaults", "", "", []string{"info", "error"}},
		{"debug", " DEBUG ", " JSON ", []string{"debug", "info", "error"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			log, err := logger.New(&out, tt.level, tt.format)
			if err != nil {
				t.Fatal(err)
			}
			log.Debug().Msg("detail")
			log.Info().Int("count", 42).Msg("started")
			log.Error().Err(errors.New("connection lost")).Msg("failed")

			lines := bytes.Split(bytes.TrimSuffix(out.Bytes(), []byte("\n")), []byte("\n"))
			if len(lines) != len(tt.levels) {
				t.Fatalf("число JSON-строк = %d, ожидалось %d", len(lines), len(tt.levels))
			}
			for i, line := range lines {
				var event struct {
					Level   string `json:"level"`
					Service string `json:"service"`
					Time    string `json:"time"`
					Message string `json:"message"`
					Count   int    `json:"count"`
					Error   string `json:"error"`
				}
				if err := json.Unmarshal(line, &event); err != nil {
					t.Fatalf("неверная JSON-строка: %v", err)
				}
				if event.Level != tt.levels[i] || event.Service != "music-api" {
					t.Fatalf("неверный уровень или сервис: %+v", event)
				}
				if _, err := time.Parse(time.RFC3339, event.Time); err != nil {
					t.Fatalf("неверная метка времени %q: %v", event.Time, err)
				}
				switch event.Level {
				case "debug":
					if event.Message != "detail" {
						t.Errorf("сообщение debug = %q", event.Message)
					}
				case "info":
					if event.Message != "started" || event.Count != 42 {
						t.Errorf("неверное событие info: %+v", event)
					}
				case "error":
					if event.Message != "failed" || event.Error != "connection lost" {
						t.Errorf("неверное событие error: %+v", event)
					}
				}
			}
		})
	}
}

func TestNewRejectsInvalidSettings(t *testing.T) {
	for _, tt := range []struct {
		name   string
		level  string
		format string
	}{
		{"level", "unknown-level", "json"},
		{"format", "info", "unknown-format"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			if _, err := logger.New(&out, tt.level, tt.format); err == nil {
				t.Fatal("конструктор принял неверную настройку")
			}
		})
	}
}

func TestNewConsoleLogger(t *testing.T) {
	var out bytes.Buffer
	log, err := logger.New(&out, "info", " CONSOLE ")
	if err != nil {
		t.Fatal(err)
	}
	log.Info().Msg("server ready")

	if !strings.Contains(out.String(), "server ready") || strings.Contains(out.String(), "\x1b") || json.Valid(out.Bytes()) {
		t.Fatalf("неверный вывод console: %q", out.String())
	}
}
