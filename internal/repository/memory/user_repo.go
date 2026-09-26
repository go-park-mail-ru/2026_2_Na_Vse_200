// Package memory реализует репозитории в памяти процесса.
// Используется до готовности слоя на PostgreSQL: данные живут до перезапуска.
package memory

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository"
)

// UserRepo хранит аккаунты в мапе под мьютексом: запросы обрабатываются
// в разных горутинах.
type UserRepo struct {
	mu sync.RWMutex

	users  map[models.ID]models.User
	byMail map[string]models.ID // индекс для поиска по email без перебора
}

// NewUserRepo создаёт пустое хранилище.
func NewUserRepo() *UserRepo {
	return &UserRepo{
		users:  make(map[models.ID]models.User),
		byMail: make(map[string]models.ID),
	}
}

// Create сохраняет пользователя, проставляя ID и CreatedAt.
// Проверка занятости email и вставка идут под одной блокировкой: иначе два
// одновременных запроса прошли бы проверку оба и создали два аккаунта.
func (s *UserRepo) Create(ctx context.Context, user models.User) (models.User, error) {
	if err := ctx.Err(); err != nil {
		return models.User{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.byMail[user.Email]; exists {
		return models.User{}, repository.ErrEmailTaken
	}

	id, err := newUserID()
	if err != nil {
		return models.User{}, err
	}
	user.ID = id
	user.CreatedAt = time.Now()

	s.users[user.ID] = user
	s.byMail[user.Email] = user.ID

	return user, nil
}

// GetByID возвращает пользователя по идентификатору.
func (s *UserRepo) GetByID(ctx context.Context, id models.ID) (models.User, error) {
	if err := ctx.Err(); err != nil {
		return models.User{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[id]
	if !ok {
		return models.User{}, repository.ErrUserNotFound
	}
	return user, nil
}

// GetByEmail возвращает пользователя вместе с хешем пароля.
// Email ожидается нормализованным.
func (s *UserRepo) GetByEmail(ctx context.Context, email string) (models.User, error) {
	if err := ctx.Err(); err != nil {
		return models.User{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.byMail[email]
	if !ok {
		return models.User{}, repository.ErrUserNotFound
	}
	return s.users[id], nil
}

func newUserID() (models.ID, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return models.ID(fmt.Sprintf("%x-%x-%x-%x-%x", raw[0:4], raw[4:6], raw[6:8], raw[8:10], raw[10:16])), nil
}
