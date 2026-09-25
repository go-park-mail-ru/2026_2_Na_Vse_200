// Package auth считает и проверяет хеши паролей.
package auth

import (
	"errors"
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"
)

// MinCost — минимальная цена bcrypt. Используется в тестах, чтобы не тратить
// на каждый пароль десятки миллисекунд.
const MinCost = bcrypt.MinCost

// Hasher считает и проверяет хеш пароля.
type Hasher interface {
	Hash(password string) (string, error)
	Verify(password, hash string) bool
}

// BcryptHasher реализует Hasher поверх bcrypt.
// Соль и цена хранятся внутри строки хеша, отдельных колонок в БД не нужно.
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

// Hash считает хеш пароля. Пароли длиннее 72 байт bcrypt не принимает.
func (h BcryptHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("хеширование пароля: %w", err)
	}
	return string(hash), nil
}

// Verify сообщает, соответствует ли пароль хешу.
// Непригодная строка хеша возвращает false и попадает в лог.
func (h BcryptHasher) Verify(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err == nil {
		return true
	}

	if !errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		log.Printf("проверка пароля: непригодный хеш: %v", err)
	}
	return false
}
