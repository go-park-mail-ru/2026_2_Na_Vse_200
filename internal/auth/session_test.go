package auth

import (
	"encoding/base64"
	"testing"
)

func TestNewSessionIDUnique(t *testing.T) {
	first, err := NewSessionID()
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	second, err := NewSessionID()
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	if first == second {
		t.Error("два вызова вернули один и тот же идентификатор")
	}
}

func TestNewSessionIDLength(t *testing.T) {
	id, err := NewSessionID()
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	raw, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil {
		t.Fatalf("идентификатор не декодируется: %v", err)
	}
	if len(raw) != sessionIDBytes {
		t.Errorf("длина = %d байт, ожидалось %d", len(raw), sessionIDBytes)
	}
}
