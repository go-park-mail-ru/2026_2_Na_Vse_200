package auth

import (
	"uuid"

	"2026_2_Na_Vse_200/internal/models"
)

// NewSessionID возвращает случайный UUID для новой сессии.
func NewSessionID() models.SessionID {
	return models.SessionID(uuid.New().String())
}
