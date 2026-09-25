// Package memory реализует хранилища в памяти процесса.
// Используется до готовности слоя на PostgreSQL: данные живут до перезапуска.
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/storage"
)

// UserStorage хранит аккаунты в мапе под мьютексом: запросы обрабатываются
// в разных горутинах.
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

// Create сохраняет пользователя, проставляя ID и CreatedAt.
// Проверка занятости email и вставка идут под одной блокировкой: иначе два
// одновременных запроса прошли бы проверку оба и создали два аккаунта.
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

// GetByEmail возвращает пользователя вместе с хешем пароля.
// Email ожидается нормализованным.
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
