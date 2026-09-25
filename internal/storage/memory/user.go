// Package memory — временные реализации хранилищ в памяти процесса.
//
// Нужны, чтобы backend работал до готовности слоя на pgx: фронтенд может
// регистрироваться и входить уже сейчас. Когда появится реализация на
// PostgreSQL, в main.go меняется одна строка — обработчики зависят
// от интерфейсов пакета storage и подмены не замечают.
//
// Плата: данные живут до перезапуска сервера.
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/storage"
)

// UserStorage хранит аккаунты в мапе.
//
// Мьютекс обязателен: каждый HTTP-запрос обрабатывается в своей горутине,
// а одновременные чтение и запись мапы в Go — это гонка, при которой
// программа может аварийно завершиться.
type UserStorage struct {
	mu sync.RWMutex

	users  map[models.ID]models.User
	byMail map[string]models.ID // индекс для поиска по email без перебора
	lastID models.ID
}

// NewUserStorage создаёт пустое хранилище.
func NewUserStorage() *UserStorage {
	return &UserStorage{
		users:  make(map[models.ID]models.User),
		byMail: make(map[string]models.ID),
	}
}

// Create сохраняет пользователя и возвращает его с проставленными ID и CreatedAt.
//
// Проверка занятости email и вставка выполняются под одной блокировкой.
// Если сначала проверить, отпустить замок и только потом вставлять,
// два одновременных запроса с одинаковым email пройдут проверку оба
// и создадут два аккаунта.
func (s *UserStorage) Create(ctx context.Context, user models.User) (models.User, error) {
	if err := ctx.Err(); err != nil {
		return models.User{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.byMail[user.Email]; exists {
		return models.User{}, storage.ErrEmailTaken
	}

	s.lastID++
	user.ID = s.lastID
	user.CreatedAt = time.Now()

	s.users[user.ID] = user
	s.byMail[user.Email] = user.ID

	return user, nil
}

// GetByID возвращает пользователя по идентификатору.
func (s *UserStorage) GetByID(ctx context.Context, id models.ID) (models.User, error) {
	if err := ctx.Err(); err != nil {
		return models.User{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[id]
	if !ok {
		return models.User{}, storage.ErrUserNotFound
	}
	return user, nil
}

// GetByEmail возвращает пользователя вместе с хешем пароля: он нужен
// при входе. Email ожидается уже нормализованным.
func (s *UserStorage) GetByEmail(ctx context.Context, email string) (models.User, error) {
	if err := ctx.Err(); err != nil {
		return models.User{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.byMail[email]
	if !ok {
		return models.User{}, storage.ErrUserNotFound
	}
	return s.users[id], nil
}
