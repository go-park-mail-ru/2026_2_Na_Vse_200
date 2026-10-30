// Package auth implements authentication independently of HTTP and storage.
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
	"2026_2_Na_Vse_200/pkg/validation"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
)

type ValidationError struct{ Fields map[string]string }

func (e *ValidationError) Error() string { return "validation failed" }

type SignupInput struct {
	Email       string
	Password    string
	DisplayName string
}

type LoginInput struct {
	Email    string
	Password string
}

type Result struct {
	User    models.User
	Session models.Session
}

// Usecase coordinates repositories and password/session adapters.
type Usecase struct {
	users        ports.UserRepository
	sessions     ports.SessionRepository
	hasher       ports.PasswordHasher
	newSessionID func() models.SessionID
	now          func() time.Time
	ttl          time.Duration
}

func New(users ports.UserRepository, sessions ports.SessionRepository, hasher ports.PasswordHasher, newSessionID func() models.SessionID, now func() time.Time, ttl time.Duration) *Usecase {
	return &Usecase{users: users, sessions: sessions, hasher: hasher, newSessionID: newSessionID, now: now, ttl: ttl}
}

func (u *Usecase) signIn(ctx context.Context, user models.User) (Result, error) {
	session := models.Session{ID: u.newSessionID(), UserID: user.ID, ExpiresAt: u.now().Add(u.ttl)}
	if err := u.sessions.Create(ctx, session); err != nil {
		return Result{}, fmt.Errorf("create session: %w", err)
	}
	return Result{User: user, Session: session}, nil
}

func (u *Usecase) Signup(ctx context.Context, in SignupInput) (Result, error) {
	form := validation.Signup(validation.SignupInput(in))
	if !form.Valid() {
		return Result{}, &ValidationError{Fields: form.Fields}
	}
	hash, err := u.hasher.Hash(form.Password)
	if err != nil {
		return Result{}, fmt.Errorf("hash password: %w", err)
	}
	user, err := u.users.Create(ctx, models.User{Email: form.Email, DisplayName: form.DisplayName, PasswordHash: hash})
	if err != nil {
		return Result{}, fmt.Errorf("create user: %w", err)
	}
	return u.signIn(ctx, user)
}

func (u *Usecase) Login(ctx context.Context, in LoginInput) (Result, error) {
	form := validation.Login(validation.LoginInput(in))
	if !form.Valid() {
		return Result{}, &ValidationError{Fields: form.Fields}
	}
	user, err := u.users.GetByEmail(ctx, form.Email)
	if errors.Is(err, models.ErrUserNotFound) {
		return Result{}, ErrInvalidCredentials
	}
	if err != nil {
		return Result{}, fmt.Errorf("find user: %w", err)
	}
	matched, err := u.hasher.Verify(form.Password, user.PasswordHash)
	if err != nil {
		return Result{}, fmt.Errorf("verify password: %w", err)
	}
	if !matched {
		return Result{}, ErrInvalidCredentials
	}
	return u.signIn(ctx, user)
}

func (u *Usecase) Me(ctx context.Context, id models.SessionID) (models.User, error) {
	if id == "" {
		return models.User{}, ErrUnauthorized
	}
	session, err := u.sessions.GetByID(ctx, id)
	if errors.Is(err, models.ErrSessionNotFound) || errors.Is(err, models.ErrSessionExpired) {
		return models.User{}, ErrUnauthorized
	}
	if err != nil {
		return models.User{}, fmt.Errorf("read session: %w", err)
	}
	if session.IsExpired(u.now()) {
		return models.User{}, ErrUnauthorized
	}
	user, err := u.users.GetByID(ctx, session.UserID)
	if errors.Is(err, models.ErrUserNotFound) {
		return models.User{}, ErrUnauthorized
	}
	if err != nil {
		return models.User{}, fmt.Errorf("find user: %w", err)
	}
	return user, nil
}

func (u *Usecase) Logout(ctx context.Context, id models.SessionID) error {
	if id == "" {
		return nil
	}
	if err := u.sessions.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
