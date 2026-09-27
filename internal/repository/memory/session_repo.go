package memory

import (
	"context"
	"sync"
	"time"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository"
)

// SessionRepo хранит сессии в мапе под мьютексом. Мьютекс обычный, а не
// RWMutex: чтение удаляет просроченную запись, то есть тоже пишет.
type SessionRepo struct {
	mu       sync.Mutex
	sessions map[string]models.Session
}

// NewSessionRepo создаёт пустое хранилище.
func NewSessionRepo() *SessionRepo {
	return &SessionRepo{sessions: make(map[string]models.Session)}
}

// Create сохраняет сессию.
func (s *SessionRepo) Create(ctx context.Context, session models.Session) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[session.ID] = session
	return nil
}

// GetByID возвращает сессию по идентификатору. Просроченная запись удаляется:
// иначе мапа растёт до перезапуска сервера.
func (s *SessionRepo) GetByID(ctx context.Context, id string) (models.Session, error) {
	if err := ctx.Err(); err != nil {
		return models.Session{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return models.Session{}, repository.ErrSessionNotFound
	}
	if session.IsExpired(time.Now()) {
		delete(s.sessions, id)
		return models.Session{}, repository.ErrSessionExpired
	}

	return session, nil
}

// Delete удаляет сессию. Удаление несуществующей ошибкой не считается.
func (s *SessionRepo) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, id)
	return nil
}
