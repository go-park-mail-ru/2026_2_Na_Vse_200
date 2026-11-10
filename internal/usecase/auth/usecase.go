package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"2026_2_Na_Vse_200/internal/logger"
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

// Usecase выполняет сценарии авторизации через переданные зависимости.
type Usecase struct {
	users        ports.UserRepository
	sessions     ports.SessionRepository
	hasher       ports.PasswordHasher
	newSessionID func() models.SessionID
	now          func() time.Time
	ttl          time.Duration
	log          *slog.Logger
}

func NewAuthUsecase(
	users ports.UserRepository,
	sessions ports.SessionRepository,
	hasher ports.PasswordHasher,
	newSessionID func() models.SessionID,
	now func() time.Time,
	ttl time.Duration,
	log *slog.Logger,
) *Usecase {
	if log == nil {
		log = logger.Discard()
	}
	return &Usecase{
		users:        users,
		sessions:     sessions,
		hasher:       hasher,
		newSessionID: newSessionID,
		now:          now,
		ttl:          ttl,
		log:          log,
	}
}

func (u *Usecase) signIn(ctx context.Context, user models.User) (Result, error) {
	log := logger.FromContext(ctx, u.log)
	session := models.Session{ID: u.newSessionID(), UserID: user.ID, ExpiresAt: u.now().Add(u.ttl)}
	if err := u.sessions.Create(ctx, session); err != nil {
		log.Error("auth.signIn: создание сессии", slog.Any("error", err), slog.String("user_id", string(user.ID)))
		return Result{}, fmt.Errorf("create session: %w", err)
	}
	log.Info("auth.signIn: сессия создана",
		slog.String("user_id", string(user.ID)),
	)
	return Result{User: user, Session: session}, nil
}

func (u *Usecase) Signup(ctx context.Context, in SignupInput) (Result, error) {
	log := logger.FromContext(ctx, u.log)
	log.Info("auth.Signup: старт")

	form := validation.Signup(validation.SignupInput(in))
	if !form.Valid() {
		log.Info("auth.Signup: ошибка валидации")
		return Result{}, &ValidationError{Fields: form.Fields}
	}
	hash, err := u.hasher.Hash(form.Password)
	if err != nil {
		log.Error("auth.Signup: хеш пароля", slog.Any("error", err))
		return Result{}, fmt.Errorf("hash password: %w", err)
	}
	user, err := u.users.Create(ctx, models.User{Email: form.Email, DisplayName: form.DisplayName, PasswordHash: hash})
	if err != nil {
		log.Error("auth.Signup: создание пользователя", slog.Any("error", err))
		return Result{}, fmt.Errorf("create user: %w", err)
	}
	log.Info("auth.Signup: пользователь создан", slog.String("user_id", string(user.ID)))
	return u.signIn(ctx, user)
}

func (u *Usecase) Login(ctx context.Context, in LoginInput) (Result, error) {
	log := logger.FromContext(ctx, u.log)
	log.Info("auth.Login: старт")

	form := validation.Login(validation.LoginInput(in))
	if !form.Valid() {
		log.Info("auth.Login: ошибка валидации")
		return Result{}, &ValidationError{Fields: form.Fields}
	}
	user, err := u.users.GetByEmail(ctx, form.Email)
	if errors.Is(err, models.ErrUserNotFound) {
		log.Info("auth.Login: пользователь не найден")
		return Result{}, ErrInvalidCredentials
	}
	if err != nil {
		log.Error("auth.Login: чтение пользователя", slog.Any("error", err))
		return Result{}, fmt.Errorf("find user: %w", err)
	}
	log.Info("auth.Login: пользователь найден", slog.String("user_id", string(user.ID)))

	matched, err := u.hasher.Verify(form.Password, user.PasswordHash)
	if err != nil {
		log.Error("auth.Login: проверка пароля", slog.Any("error", err), slog.String("user_id", string(user.ID)))
		return Result{}, fmt.Errorf("verify password: %w", err)
	}
	if !matched {
		log.Info("auth.Login: неверный пароль", slog.String("user_id", string(user.ID)))
		return Result{}, ErrInvalidCredentials
	}
	return u.signIn(ctx, user)
}

func (u *Usecase) SessionByID(ctx context.Context, id models.SessionID) (models.Session, error) {
	log := logger.FromContext(ctx, u.log)
	if id == "" {
		return models.Session{}, models.ErrSessionNotFound
	}
	session, err := u.sessions.GetByID(ctx, id)
	if errors.Is(err, models.ErrSessionNotFound) || errors.Is(err, models.ErrSessionExpired) {
		log.Info("auth.SessionByID: сессия недоступна", slog.Any("error", err))
		return models.Session{}, err
	}
	if err != nil {
		log.Error("auth.SessionByID: чтение сессии", slog.Any("error", err))
		return models.Session{}, fmt.Errorf("read session: %w", err)
	}
	if session.IsExpired(u.now()) {
		log.Info("auth.SessionByID: сессия истекла")
		return models.Session{}, models.ErrSessionExpired
	}
	return session, nil
}

func (u *Usecase) Me(ctx context.Context, id models.SessionID) (models.User, error) {
	log := logger.FromContext(ctx, u.log)
	log.Info("auth.Me: старт")

	session, err := u.SessionByID(ctx, id)
	if errors.Is(err, models.ErrSessionNotFound) || errors.Is(err, models.ErrSessionExpired) {
		return models.User{}, ErrUnauthorized
	}
	if err != nil {
		return models.User{}, err
	}
	user, err := u.users.GetByID(ctx, session.UserID)
	if errors.Is(err, models.ErrUserNotFound) {
		log.Info("auth.Me: пользователь не найден", slog.String("user_id", string(session.UserID)))
		return models.User{}, ErrUnauthorized
	}
	if err != nil {
		log.Error("auth.Me: чтение пользователя", slog.Any("error", err), slog.String("user_id", string(session.UserID)))
		return models.User{}, fmt.Errorf("find user: %w", err)
	}
	log.Info("auth.Me: успех", slog.String("user_id", string(user.ID)))
	return user, nil
}

func (u *Usecase) Logout(ctx context.Context, id models.SessionID) error {
	log := logger.FromContext(ctx, u.log)
	log.Info("auth.Logout: старт")
	if id == "" {
		return nil
	}
	if err := u.sessions.Delete(ctx, id); err != nil {
		log.Error("auth.Logout: удаление сессии", slog.Any("error", err))
		return fmt.Errorf("delete session: %w", err)
	}
	log.Info("auth.Logout: сессия удалена")
	return nil
}
