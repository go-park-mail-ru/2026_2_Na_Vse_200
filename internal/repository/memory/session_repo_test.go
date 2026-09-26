package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository"
)

func TestSessionCreateAndGet(t *testing.T) {
	ctx := context.Background()
	repo := NewSessionRepo()

	session := models.Session{
		ID:        "kR3n8Qv1XpLmA7bYcZfTdWgHsJuNeOiP",
		UserID:    "10000000-0000-4000-8000-000000000007",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	if err := repo.Create(ctx, session); err != nil {
		t.Fatalf("Create: неожиданная ошибка: %v", err)
	}

	got, err := repo.GetByID(ctx, session.ID)
	if err != nil {
		t.Fatalf("GetByID: неожиданная ошибка: %v", err)
	}

	if got.UserID != session.UserID {
		t.Errorf("GetByID вернул UserID %s, ожидался %s", got.UserID, session.UserID)
	}
}

func TestSessionNotFound(t *testing.T) {
	repo := NewSessionRepo()

	_, err := repo.GetByID(context.Background(), "чужой идентификатор")
	if !errors.Is(err, repository.ErrSessionNotFound) {
		t.Errorf("ошибка = %v, ожидалась ErrSessionNotFound", err)
	}
}

func TestSessionExpired(t *testing.T) {
	ctx := context.Background()
	repo := NewSessionRepo()

	session := models.Session{
		ID:        "kR3n8Qv1XpLmA7bYcZfTdWgHsJuNeOiP",
		UserID:    "10000000-0000-4000-8000-000000000007",
		ExpiresAt: time.Now().Add(-time.Minute),
	}

	if err := repo.Create(ctx, session); err != nil {
		t.Fatalf("Create: неожиданная ошибка: %v", err)
	}

	if _, err := repo.GetByID(ctx, session.ID); !errors.Is(err, repository.ErrSessionExpired) {
		t.Fatalf("ошибка = %v, ожидалась ErrSessionExpired", err)
	}

	// Просроченная запись должна исчезнуть при чтении, иначе мапа растёт.
	if _, err := repo.GetByID(ctx, session.ID); !errors.Is(err, repository.ErrSessionNotFound) {
		t.Errorf("повторное чтение: ошибка = %v, ожидалась ErrSessionNotFound", err)
	}
}

func TestSessionDelete(t *testing.T) {
	ctx := context.Background()
	repo := NewSessionRepo()

	session := models.Session{
		ID:        "kR3n8Qv1XpLmA7bYcZfTdWgHsJuNeOiP",
		UserID:    "10000000-0000-4000-8000-000000000007",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	if err := repo.Create(ctx, session); err != nil {
		t.Fatalf("Create: неожиданная ошибка: %v", err)
	}

	if err := repo.Delete(ctx, session.ID); err != nil {
		t.Fatalf("Delete: неожиданная ошибка: %v", err)
	}

	if _, err := repo.GetByID(ctx, session.ID); !errors.Is(err, repository.ErrSessionNotFound) {
		t.Errorf("после удаления ошибка = %v, ожидалась ErrSessionNotFound", err)
	}

	// Повторный выход не должен отличаться от первого.
	if err := repo.Delete(ctx, session.ID); err != nil {
		t.Errorf("удаление несуществующей сессии вернуло ошибку: %v", err)
	}
}

func TestSessionCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	repo := NewSessionRepo()
	if err := repo.Create(ctx, models.Session{ID: "kR3n8Qv1XpLmA7bYcZfTdWgHsJuNeOiP"}); err == nil {
		t.Error("Create с отменённым контекстом должен возвращать ошибку")
	}
}
