package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// sessionIDBytes — длина идентификатора до кодирования.
const sessionIDBytes = 32

// NewSessionID возвращает случайный идентификатор сессии.
func NewSessionID() (string, error) {
	buf := make([]byte, sessionIDBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("генерация идентификатора сессии: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}
