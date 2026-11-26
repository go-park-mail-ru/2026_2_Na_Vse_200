package sessions

import (
	"context"
	"errors"
	"time"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
	"github.com/rs/zerolog"
)

type SessionManager struct {
	repo  ports.SessionRepository
	newID func() models.SessionID
	now   func() time.Time
	ttl   time.Duration
}

func NewSessionManager(
	repo ports.SessionRepository,
	newID func() models.SessionID,
	now func() time.Time,
	ttl time.Duration,
) *SessionManager {
	return &SessionManager{repo: repo, newID: newID, now: now, ttl: ttl}
}

func (s *SessionManager) CreateSession(ctx context.Context, userID models.UserID) (models.Session, error) {
	session := models.Session{ID: s.newID(), UserID: userID, ExpiresAt: s.now().Add(s.ttl)}
	if err := s.repo.Create(ctx, session); err != nil {
		logResult(ctx, "sessions.create", err)
		return models.Session{}, err
	}
	logResult(ctx, "sessions.create", nil)
	return session, nil
}

func (s *SessionManager) GetSession(ctx context.Context, id models.SessionID) (models.Session, error) {
	session, err := s.getSession(ctx, id)
	logResult(ctx, "sessions.get", err)
	return session, err
}

func (s *SessionManager) getSession(ctx context.Context, id models.SessionID) (models.Session, error) {
	if id == "" {
		return models.Session{}, models.ErrSessionNotFound
	}
	session, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return models.Session{}, err
	}
	if session.IsExpired(s.now()) {
		return models.Session{}, models.ErrSessionExpired
	}
	return session, nil
}

func (s *SessionManager) DeleteSession(ctx context.Context, id models.SessionID) error {
	var err error
	if id != "" {
		err = s.repo.Delete(ctx, id)
	}
	logResult(ctx, "sessions.delete", err)
	return err
}

func (s *SessionManager) CheckProfileAccess(ctx context.Context, id models.SessionID, target models.UserID) error {
	session, err := s.getSession(ctx, id)
	if err == nil && session.UserID != target {
		err = models.ErrForbidden
	}
	logResult(ctx, "sessions.check_profile_access", err)
	return err
}

func logResult(ctx context.Context, operation string, err error) {
	log := zerolog.Ctx(ctx).With().Str("layer", "usecase").Str("operation", operation).Logger()
	const message = "операция сессии завершена"
	switch {
	case err == nil:
		log.Info().Str("outcome", "ok").Msg(message)
	case errors.Is(err, models.ErrSessionNotFound):
		log.Debug().Str("outcome", "denied").Str("reason", "session_not_found").Msg(message)
	case errors.Is(err, models.ErrSessionExpired):
		log.Debug().Str("outcome", "denied").Str("reason", "session_expired").Msg(message)
	case errors.Is(err, models.ErrForbidden):
		log.Debug().Str("outcome", "denied").Str("reason", "profile_forbidden").Msg(message)
	case errors.Is(err, context.Canceled):
		log.Debug().Str("outcome", "canceled").Msg(message)
	case errors.Is(err, context.DeadlineExceeded):
		log.Warn().Str("outcome", "deadline_exceeded").Msg(message)
	default:
		log.Error().Str("outcome", "error").Str("reason", "repository_error").Msg(message)
	}
}
