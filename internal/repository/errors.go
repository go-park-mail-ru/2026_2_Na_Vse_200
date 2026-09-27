package repository

import "errors"

// Ошибки, которые обработчик различает при выборе кода ответа.
// Проверяются через errors.Is: реализация вправе обернуть их контекстом.
var (
	ErrUserNotFound    = errors.New("user not found")
	ErrEmailTaken      = errors.New("email already taken")
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
)
