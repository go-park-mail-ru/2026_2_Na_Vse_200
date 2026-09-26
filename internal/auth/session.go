package auth

import "uuid"

// NewSessionID возвращает идентификатор сессии — UUID версии 4 по RFC 9562.
// Пакет uuid берёт случайность из crypto/rand, поэтому значение не подобрать.
func NewSessionID() string {
	return uuid.New().String()
}
