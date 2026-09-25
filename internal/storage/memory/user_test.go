package memory

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/storage"
)

func TestCreateAndGet(t *testing.T) {
	ctx := context.Background()
	store := NewUserStorage()

	created, err := store.Create(ctx, models.User{
		Email:        "andrey@example.com",
		PasswordHash: "sha256$c29sdA$aGFzaA",
		DisplayName:  "Андрей",
	})
	if err != nil {
		t.Fatalf("Create: неожиданная ошибка: %v", err)
	}

	if created.ID == 0 {
		t.Error("Create не проставил ID")
	}
	if created.CreatedAt.IsZero() {
		t.Error("Create не проставил CreatedAt")
	}

	byID, err := store.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: неожиданная ошибка: %v", err)
	}
	if byID.Email != created.Email {
		t.Errorf("GetByID вернул email %q, ожидался %q", byID.Email, created.Email)
	}

	byEmail, err := store.GetByEmail(ctx, "andrey@example.com")
	if err != nil {
		t.Fatalf("GetByEmail: неожиданная ошибка: %v", err)
	}
	// Хеш нужен при входе, поэтому хранилище обязано его возвращать.
	if byEmail.PasswordHash != created.PasswordHash {
		t.Error("GetByEmail не вернул хеш пароля, он нужен для проверки при входе")
	}
}

func TestGetNotFound(t *testing.T) {
	ctx := context.Background()
	store := NewUserStorage()

	// Ошибку проверяем через errors.Is, а не сравнением значений:
	// реализация может обернуть её контекстом.
	if _, err := store.GetByID(ctx, 42); !errors.Is(err, storage.ErrUserNotFound) {
		t.Errorf("GetByID: ошибка = %v, ожидалась ErrUserNotFound", err)
	}
	if _, err := store.GetByEmail(ctx, "нет@такого.com"); !errors.Is(err, storage.ErrUserNotFound) {
		t.Errorf("GetByEmail: ошибка = %v, ожидалась ErrUserNotFound", err)
	}
}

func TestCreateDuplicateEmail(t *testing.T) {
	ctx := context.Background()
	store := NewUserStorage()

	user := models.User{Email: "andrey@example.com", DisplayName: "Андрей"}
	if _, err := store.Create(ctx, user); err != nil {
		t.Fatalf("первое создание: неожиданная ошибка: %v", err)
	}

	if _, err := store.Create(ctx, user); !errors.Is(err, storage.ErrEmailTaken) {
		t.Errorf("повторное создание: ошибка = %v, ожидалась ErrEmailTaken", err)
	}
}

// Ключевая проверка: одновременные запросы с одним email не должны создать
// два аккаунта. Проверка занятости и вставка обязаны идти под одной блокировкой.
func TestCreateConcurrentSameEmail(t *testing.T) {
	ctx := context.Background()
	store := NewUserStorage()

	const attempts = 50

	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		succeeded int
	)

	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			_, err := store.Create(ctx, models.User{
				Email:       "andrey@example.com",
				DisplayName: "Андрей",
			})
			if err == nil {
				mu.Lock()
				succeeded++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if succeeded != 1 {
		t.Errorf("успешных регистраций = %d, ожидалась ровно 1", succeeded)
	}
}

// Контекст уже отменён — клиент ушёл, работать незачем.
func TestCreateCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	store := NewUserStorage()
	if _, err := store.Create(ctx, models.User{Email: "andrey@example.com"}); err == nil {
		t.Error("Create с отменённым контекстом должен возвращать ошибку")
	}
}
