package memory

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository"
)

func TestCreateAndGet(t *testing.T) {
	ctx := context.Background()
	repo := NewUserRepo()

	created, err := repo.Create(ctx, models.User{
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

	byID, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: неожиданная ошибка: %v", err)
	}
	if byID.Email != created.Email {
		t.Errorf("GetByID вернул email %q, ожидался %q", byID.Email, created.Email)
	}

	byEmail, err := repo.GetByEmail(ctx, "andrey@example.com")
	if err != nil {
		t.Fatalf("GetByEmail: неожиданная ошибка: %v", err)
	}
	if byEmail.PasswordHash != created.PasswordHash {
		t.Error("GetByEmail не вернул хеш пароля, он нужен для проверки при входе")
	}
}

func TestGetNotFound(t *testing.T) {
	ctx := context.Background()
	repo := NewUserRepo()

	if _, err := repo.GetByID(ctx, 42); !errors.Is(err, repository.ErrUserNotFound) {
		t.Errorf("GetByID: ошибка = %v, ожидалась ErrUserNotFound", err)
	}
	if _, err := repo.GetByEmail(ctx, "нет@такого.com"); !errors.Is(err, repository.ErrUserNotFound) {
		t.Errorf("GetByEmail: ошибка = %v, ожидалась ErrUserNotFound", err)
	}
}

func TestCreateDuplicateEmail(t *testing.T) {
	ctx := context.Background()
	repo := NewUserRepo()

	user := models.User{Email: "andrey@example.com", DisplayName: "Андрей"}
	if _, err := repo.Create(ctx, user); err != nil {
		t.Fatalf("первое создание: неожиданная ошибка: %v", err)
	}

	if _, err := repo.Create(ctx, user); !errors.Is(err, repository.ErrEmailTaken) {
		t.Errorf("повторное создание: ошибка = %v, ожидалась ErrEmailTaken", err)
	}
}

// Одновременные запросы с одним email не должны создать два аккаунта.
func TestCreateConcurrentSameEmail(t *testing.T) {
	ctx := context.Background()
	repo := NewUserRepo()

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

			_, err := repo.Create(ctx, models.User{
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

func TestCreateCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	repo := NewUserRepo()
	if _, err := repo.Create(ctx, models.User{Email: "andrey@example.com"}); err == nil {
		t.Error("Create с отменённым контекстом должен возвращать ошибку")
	}
}
