// Package auth отвечает за хеширование и проверку паролей.
//
// Алгоритм спрятан за интерфейсом Hasher: обработчики знают только «посчитать хеш»
// и «сверить пароль с хешем». Благодаря этому тесты обработчиков подставляют
// дешёвый хешер, а не гоняют настоящий bcrypt на каждый запрос.
package auth

import (
	"errors"
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"
)

// Hasher считает и проверяет хеш пароля.
type Hasher interface {
	// Hash возвращает строку, которую хранит БД в колонке account.password_hash.
	Hash(password string) (string, error)

	// Verify сообщает, соответствует ли пароль хешу. Непригодный хеш —
	// это не паника и не ошибка, а просто «не подходит».
	Verify(password, hash string) bool
}

// BcryptHasher — реализация на bcrypt из golang.org/x/crypto.
//
// bcrypt создан специально для паролей и **намеренно медленный**: внутри
// многократное повторение вычислений, число проходов задаётся параметром cost.
// Обычный sha256 быстрый, и утёкшую базу перебирают на видеокарте миллиардами
// вариантов в секунду; bcrypt снижает скорость перебора до десятков в секунду.
//
// Соль bcrypt генерирует сам и хранит внутри строки хеша, поэтому отдельная
// колонка под неё не нужна. Формат: $2a$<cost>$<22 символа соли><31 символ хеша>.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher создаёт хешер с ценой по умолчанию (bcrypt.DefaultCost = 10).
// Цену можно поднимать по мере удешевления железа: каждая единица удваивает
// время вычисления.
func NewBcryptHasher() BcryptHasher {
	return BcryptHasher{cost: bcrypt.DefaultCost}
}

// MinCost — минимальная допустимая цена. Нужна тестам, чтобы не тратить
// по 50 мс на каждый пароль. В рабочем коде не используется.
const MinCost = bcrypt.MinCost

// NewBcryptHasherWithCost создаёт хешер с заданной ценой.
// Нужен тестам: с bcrypt.MinCost они не тратят по 50 мс на каждый пароль.
// В рабочем коде используется NewBcryptHasher.
func NewBcryptHasherWithCost(cost int) BcryptHasher {
	return BcryptHasher{cost: cost}
}

// Hash считает хеш пароля.
//
// Пароль длиннее 72 байт bcrypt не принимает — именно поэтому верхняя граница
// в правилах валидации равна 72 (см. docs/api.md, раздел 5).
func (h BcryptHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("хеширование пароля: %w", err)
	}
	return string(hash), nil
}

// Verify сверяет пароль с хешем.
//
// CompareHashAndPassword сравнивает за постоянное время: обычное сравнение
// завершалось бы на первом несовпавшем байте, и по времени ответа хеш можно
// было бы подбирать по байту за раз.
func (h BcryptHasher) Verify(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err == nil {
		return true
	}

	// Пароль не подошёл — обычный случай, молчим. Любая другая ошибка означает
	// испорченную строку хеша в базе: это уже повод заглянуть в логи,
	// но клиенту в обоих случаях просто не даём войти.
	if !errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		log.Printf("проверка пароля: непригодный хеш: %v", err)
	}
	return false
}
