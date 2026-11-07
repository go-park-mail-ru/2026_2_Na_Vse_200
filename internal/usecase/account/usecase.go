package account

import (
	"context"
	"errors"
	"strings"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
	"2026_2_Na_Vse_200/internal/validation"
)

type Repository interface {
	ChangeEmail(context.Context, models.UserID, string, string) (models.User, error)
	ChangePassword(context.Context, models.UserID, string, string) error
	DeleteAccount(context.Context, models.UserID, string) error
}

type Usecase struct {
	users    ports.UserRepository
	accounts Repository
	hasher   ports.PasswordHasher
}

func New(users ports.UserRepository, accounts Repository, hasher ports.PasswordHasher) *Usecase {
	return &Usecase{users: users, accounts: accounts, hasher: hasher}
}

func (u *Usecase) verify(ctx context.Context, owner, id models.UserID, password string) (models.User, error) {
	if owner != id {
		return models.User{}, models.ErrForbidden
	}
	if strings.TrimSpace(password) == "" {
		return models.User{}, &models.ValidationError{Fields: map[string]string{"current_password": "Введите текущий пароль"}}
	}
	user, err := u.users.GetByID(ctx, id)
	if errors.Is(err, models.ErrUserNotFound) {
		return models.User{}, models.ErrUnauthorized
	}
	if err != nil {
		return models.User{}, err
	}
	valid, err := u.hasher.Verify(strings.TrimSpace(password), user.PasswordHash)
	if err != nil {
		return models.User{}, err
	}
	if !valid {
		return models.User{}, models.ErrCurrentPassword
	}
	return user, nil
}

func (u *Usecase) ChangeEmail(ctx context.Context, owner, id models.UserID, password, email string) (models.User, error) {
	user, err := u.verify(ctx, owner, id, password)
	if err != nil {
		return models.User{}, err
	}
	form := validation.ProfileUpdate(validation.ProfileUpdateInput{Email: email, DisplayName: user.DisplayName})
	if message, ok := form.Fields["email"]; ok {
		return models.User{}, &models.ValidationError{Fields: map[string]string{"email": message}}
	}
	return u.accounts.ChangeEmail(ctx, id, user.PasswordHash, form.Email)
}

func (u *Usecase) ChangePassword(ctx context.Context, owner, id models.UserID, password, next string) error {
	user, err := u.verify(ctx, owner, id, password)
	if err != nil {
		return err
	}
	form := validation.Signup(validation.SignupInput{Password: next})
	if message, ok := form.Fields["password"]; ok {
		return &models.ValidationError{Fields: map[string]string{"new_password": message}}
	}
	hash, err := u.hasher.Hash(form.Password)
	if err != nil {
		return err
	}
	return u.accounts.ChangePassword(ctx, id, user.PasswordHash, hash)
}

func (u *Usecase) Delete(ctx context.Context, owner, id models.UserID, password string) error {
	user, err := u.verify(ctx, owner, id, password)
	if err != nil {
		return err
	}
	if string(id) == "10000000-0000-4000-8000-000000000001" {
		return models.ErrForbidden
	}
	return u.accounts.DeleteAccount(ctx, id, user.PasswordHash)
}
