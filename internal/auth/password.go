// Package auth считает и проверяет хеши паролей и выдаёт идентификаторы сессий.
package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// MinCost — минимальная цена bcrypt.
const MinCost = bcrypt.MinCost

// MaxPasswordBytes — предел bcrypt: всё после 72 байт он отбрасывает.
const MaxPasswordBytes = 72

// ErrPasswordTooLong возвращается, когда пароль длиннее MaxPasswordBytes.
// Отличается от внутренних ошибок: виноват клиент, а не сервис.
var ErrPasswordTooLong = errors.New("password too long")

// Hasher считает и проверяет хеш пароля.
type Hasher interface {
	Hash(password string) (string, error)
	Verify(password, hash string) (bool, error)
}

// BcryptHasher реализует Hasher поверх bcrypt.
// Соль и цена хранятся внутри строки хеша.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher создаёт хешер с ценой по умолчанию.
func NewBcryptHasher() BcryptHasher {
	return BcryptHasher{cost: bcrypt.DefaultCost}
}

// NewBcryptHasherWithCost создаёт хешер с заданной ценой.
func NewBcryptHasherWithCost(cost int) BcryptHasher {
	return BcryptHasher{cost: cost}
}

// Hash считает хеш пароля. Пароль длиннее MaxPasswordBytes не принимается:
// обрезать нельзя, иначе проверялось бы только его начало.
func (h BcryptHasher) Hash(password string) (string, error) {
	if len(password) > MaxPasswordBytes {
		return "", ErrPasswordTooLong
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("хеширование пароля: %w", err)
	}
	return string(hash), nil
}

// Verify сообщает, соответствует ли пароль хешу.
func (h BcryptHasher) Verify(password, hash string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))

	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
		return false, nil
	default:
		return false, fmt.Errorf("проверка пароля: %w", err)
	}
}
