package sessions

import (
	"uuid"
)

// NewSessionID возвращает случайный UUID для новой сессии.
func NewSessionID() SessionID {
	return SessionID(uuid.New().String())
}
