package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	// MinCost задаёт минимальную стоимость bcrypt.
	MinCost = bcrypt.MinCost
	// MaxPasswordBytes задаёт максимальную длину пароля в байтах для bcrypt.
	MaxPasswordBytes = 72
)

// ErrPasswordTooLong означает превышение предела длины пароля для bcrypt.
var ErrPasswordTooLong = errors.New("password too long")

// BcryptHasher вычисляет и проверяет хеши bcrypt.
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

// Hash возвращает хеш пароля. Пароли длиннее MaxPasswordBytes отклоняются.
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

// Verify возвращает false для неверного пароля и ошибку для некорректного хеша.
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
