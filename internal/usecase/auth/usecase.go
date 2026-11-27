package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
	"2026_2_Na_Vse_200/internal/validation"
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

//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=mocks/mock_sessions.go -package=mocks . Sessions

// Sessions управляет сессиями для HTTP-сценариев входа и выхода.
type Sessions interface {
	CreateSession(ctx context.Context, userID models.UserID, credentialVersion int64) (models.Session, error)
	GetSession(context.Context, models.SessionID) (models.Session, error)
	DeleteSession(context.Context, models.SessionID) error
}

// Usecase выполняет сценарии авторизации через переданные зависимости.
type Usecase struct {
	users    ports.UserRepository
	sessions Sessions
	hasher   ports.PasswordHasher
}

func NewAuthUsecase(
	users ports.UserRepository,
	sessions Sessions,
	hasher ports.PasswordHasher,
) *Usecase {
	return &Usecase{users: users, sessions: sessions, hasher: hasher}
}

func (u *Usecase) signIn(ctx context.Context, user models.User) (Result, error) {
	log := zerolog.Ctx(ctx)
	log.Info().Str("layer", "usecase").Str("operation", "auth.sign_in").Msg("начало сценария")
	session, err := u.sessions.CreateSession(ctx, user.ID, user.CredentialVersion)
	if errors.Is(err, ErrInvalidCredentials) {
		log.Info().Str("user_id", string(user.ID)).Msg("auth.signIn: учётные данные изменились")
		return Result{}, ErrInvalidCredentials
	}
	if err != nil {
		log.Error().Err(err).Str("user_id", string(user.ID)).Msg("auth.signIn: создание сессии")
		return Result{}, fmt.Errorf("create session: %w", err)
	}
	log.Info().Str("user_id", string(user.ID)).Msg("auth.signIn: сессия создана")
	return Result{User: user, Session: session}, nil
}

func (u *Usecase) Signup(ctx context.Context, in SignupInput) (Result, error) {
	log := zerolog.Ctx(ctx)
	log.Info().Str("layer", "usecase").Str("operation", "auth.signup").Msg("начало сценария")

	form := validation.Signup(validation.SignupInput(in))
	if !form.Valid() {
		log.Info().Msg("auth.Signup: ошибка валидации")
		return Result{}, &ValidationError{Fields: form.Fields}
	}
	hash, err := u.hasher.Hash(form.Password)
	if err != nil {
		log.Error().Err(err).Msg("auth.Signup: хеш пароля")
		return Result{}, fmt.Errorf("hash password: %w", err)
	}
	user, err := u.users.Create(ctx, models.User{Email: form.Email, DisplayName: form.DisplayName, PasswordHash: hash})
	if err != nil {
		log.Error().Err(err).Msg("auth.Signup: создание пользователя")
		return Result{}, fmt.Errorf("create user: %w", err)
	}
	log.Info().Str("user_id", string(user.ID)).Msg("auth.Signup: пользователь создан")
	return u.signIn(ctx, user)
}

func (u *Usecase) Login(ctx context.Context, in LoginInput) (Result, error) {
	log := zerolog.Ctx(ctx)
	log.Info().Str("layer", "usecase").Str("operation", "auth.login").Msg("начало сценария")

	form := validation.Login(validation.LoginInput(in))
	if !form.Valid() {
		log.Info().Msg("auth.Login: ошибка валидации")
		return Result{}, &ValidationError{Fields: form.Fields}
	}
	user, err := u.users.GetByEmail(ctx, form.Email)
	if errors.Is(err, models.ErrUserNotFound) {
		log.Info().Msg("auth.Login: пользователь не найден")
		return Result{}, ErrInvalidCredentials
	}
	if err != nil {
		log.Error().Err(err).Msg("auth.Login: чтение пользователя")
		return Result{}, fmt.Errorf("find user: %w", err)
	}
	log.Info().Str("user_id", string(user.ID)).Msg("auth.Login: пользователь найден")

	matched, err := u.hasher.Verify(form.Password, user.PasswordHash)
	if err != nil {
		log.Error().Err(err).Str("user_id", string(user.ID)).Msg("auth.Login: проверка пароля")
		return Result{}, fmt.Errorf("verify password: %w", err)
	}
	if !matched {
		log.Info().Str("user_id", string(user.ID)).Msg("auth.Login: неверный пароль")
		return Result{}, ErrInvalidCredentials
	}
	return u.signIn(ctx, user)
}

func (u *Usecase) SessionByID(ctx context.Context, id models.SessionID) (models.Session, error) {
	log := zerolog.Ctx(ctx)
	log.Info().Str("layer", "usecase").Str("operation", "auth.session_by_id").Msg("начало сценария")
	session, err := u.sessions.GetSession(ctx, id)
	if errors.Is(err, models.ErrSessionNotFound) || errors.Is(err, models.ErrSessionExpired) {
		log.Info().Err(err).Msg("auth.SessionByID: сессия недоступна")
		return models.Session{}, err
	}
	if err != nil {
		log.Error().Err(err).Msg("auth.SessionByID: чтение сессии")
		return models.Session{}, fmt.Errorf("read session: %w", err)
	}
	return session, nil
}

func (u *Usecase) Me(ctx context.Context, id models.SessionID) (models.User, error) {
	log := zerolog.Ctx(ctx)
	log.Info().Str("layer", "usecase").Str("operation", "auth.me").Msg("начало сценария")

	session, err := u.SessionByID(ctx, id)
	if errors.Is(err, models.ErrSessionNotFound) || errors.Is(err, models.ErrSessionExpired) {
		return models.User{}, ErrUnauthorized
	}
	if err != nil {
		return models.User{}, err
	}
	user, err := u.users.GetByID(ctx, session.UserID)
	if errors.Is(err, models.ErrUserNotFound) {
		log.Info().Str("user_id", string(session.UserID)).Msg("auth.Me: пользователь не найден")
		return models.User{}, ErrUnauthorized
	}
	if err != nil {
		log.Error().Err(err).Str("user_id", string(session.UserID)).Msg("auth.Me: чтение пользователя")
		return models.User{}, fmt.Errorf("find user: %w", err)
	}
	log.Info().Str("user_id", string(user.ID)).Msg("auth.Me: успех")
	return user, nil
}

func (u *Usecase) Logout(ctx context.Context, id models.SessionID) error {
	log := zerolog.Ctx(ctx)
	log.Info().Str("layer", "usecase").Str("operation", "auth.logout").Msg("начало сценария")
	if id == "" {
		return nil
	}
	if err := u.sessions.DeleteSession(ctx, id); err != nil {
		log.Error().Err(err).Msg("auth.Logout: удаление сессии")
		return fmt.Errorf("delete session: %w", err)
	}
	log.Info().Msg("auth.Logout: сессия удалена")
	return nil
}
