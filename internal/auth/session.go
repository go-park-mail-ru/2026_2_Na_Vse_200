package auth

import (
	"uuid"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
)

// NewSessionID возвращает идентификатор сессии — UUID версии 4 по RFC 9562.
// Пакет uuid берёт случайность из crypto/rand, поэтому значение не подобрать.
func NewSessionID() models.SessionID {
	return models.SessionID(uuid.New().String())
}
