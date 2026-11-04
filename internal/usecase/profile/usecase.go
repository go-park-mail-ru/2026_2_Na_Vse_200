package profile

import (
	"context"
	"errors"
	"fmt"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
)

var (
	ErrForbidden    = errors.New("forbidden")
	ErrUnauthorized = errors.New("unauthorized")
)

type Usecase struct {
	users ports.UserRepository
}

func New(users ports.UserRepository) *Usecase {
	return &Usecase{users: users}
}

func (u *Usecase) Get(ctx context.Context, sessionUserID, requestedID models.UserID) (models.User, error) {
	if sessionUserID != requestedID {
		return models.User{}, ErrForbidden
	}

	user, err := u.users.GetByID(ctx, sessionUserID)
	if errors.Is(err, models.ErrUserNotFound) {
		return models.User{}, ErrUnauthorized
	}
	if err != nil {
		return models.User{}, fmt.Errorf("find user: %w", err)
	}
	return user, nil
}
