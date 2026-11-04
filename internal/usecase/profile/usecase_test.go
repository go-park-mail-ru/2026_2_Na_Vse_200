package profile

import (
	"context"
	"errors"
	"testing"

	"2026_2_Na_Vse_200/internal/models"
)

type userRepoStub struct {
	user    models.User
	err     error
	calls   int
	lastID  models.UserID
	failOn  bool
}

func (s *userRepoStub) Create(context.Context, models.User) (models.User, error) {
	return models.User{}, nil
}
func (s *userRepoStub) GetByEmail(context.Context, string) (models.User, error) {
	return models.User{}, nil
}
func (s *userRepoStub) GetByID(_ context.Context, id models.UserID) (models.User, error) {
	s.calls++
	s.lastID = id
	if s.failOn {
		return models.User{}, s.err
	}
	return s.user, s.err
}

func TestGet(t *testing.T) {
	owner := models.UserID("11111111-1111-1111-1111-111111111111")
	other := models.UserID("22222222-2222-2222-2222-222222222222")
	user := models.User{ID: owner, Email: "andrey@example.com", DisplayName: "Андрей"}

	t.Run("чужой id", func(t *testing.T) {
		repo := &userRepoStub{user: user}
		_, err := New(repo).Get(context.Background(), owner, other)
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("err = %v, ожидался ErrForbidden", err)
		}
		if repo.calls != 0 {
			t.Fatalf("репозиторий вызван %d раз, ожидалось 0", repo.calls)
		}
	})

	t.Run("владелец", func(t *testing.T) {
		repo := &userRepoStub{user: user}
		got, err := New(repo).Get(context.Background(), owner, owner)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if got.Email != user.Email {
			t.Fatalf("email = %q", got.Email)
		}
	})

	t.Run("пользователь удалён", func(t *testing.T) {
		repo := &userRepoStub{err: models.ErrUserNotFound, failOn: true}
		_, err := New(repo).Get(context.Background(), owner, owner)
		if !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("err = %v, ожидался ErrUnauthorized", err)
		}
	})
}
