package sessions

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog"
)

type SessionManager struct {
	repo  Repository
	newID func() SessionID
	now   func() time.Time
	ttl   time.Duration
}

func NewSessionManager(
	repo Repository,
	newID func() SessionID,
	now func() time.Time,
	ttl time.Duration,
) *SessionManager {
	return &SessionManager{repo: repo, newID: newID, now: now, ttl: ttl}
}

func (s *SessionManager) CreateSession(ctx context.Context, userID UserID, credentialVersion int64) (Session, error) {
	session := Session{
		ID: s.newID(), UserID: userID, ExpiresAt: s.now().Add(s.ttl),
		CredentialVersion: credentialVersion,
	}
	if err := s.repo.Create(ctx, session); err != nil {
		logResult(ctx, "sessions.create", err)
		return Session{}, err
	}
	logResult(ctx, "sessions.create", nil)
	return session, nil
}

func (s *SessionManager) GetSession(ctx context.Context, id SessionID) (Session, error) {
	session, err := s.getSession(ctx, id)
	logResult(ctx, "sessions.get", err)
	return session, err
}

func (s *SessionManager) getSession(ctx context.Context, id SessionID) (Session, error) {
	if id == "" {
		return Session{}, ErrSessionNotFound
	}
	session, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Session{}, err
	}
	if session.IsExpired(s.now()) {
		return Session{}, ErrSessionExpired
	}
	return session, nil
}

func (s *SessionManager) DeleteSession(ctx context.Context, id SessionID) error {
	var err error
	if id != "" {
		err = s.repo.Delete(ctx, id)
	}
	logResult(ctx, "sessions.delete", err)
	return err
}

func (s *SessionManager) CheckProfileAccess(ctx context.Context, id SessionID, target UserID) error {
	session, err := s.getSession(ctx, id)
	if err == nil && session.UserID != target {
		err = ErrForbidden
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
	case errors.Is(err, ErrSessionNotFound):
		log.Debug().Str("outcome", "denied").Str("reason", "session_not_found").Msg(message)
	case errors.Is(err, ErrSessionExpired):
		log.Debug().Str("outcome", "denied").Str("reason", "session_expired").Msg(message)
	case errors.Is(err, ErrSessionRevoked):
		log.Debug().Str("outcome", "denied").Str("reason", "session_revoked").Msg(message)
	case errors.Is(err, ErrCredentialsChanged):
		log.Debug().Str("outcome", "denied").Str("reason", "credentials_changed").Msg(message)
	case errors.Is(err, ErrForbidden):
		log.Debug().Str("outcome", "denied").Str("reason", "profile_forbidden").Msg(message)
	case errors.Is(err, context.Canceled):
		log.Debug().Str("outcome", "canceled").Msg(message)
	case errors.Is(err, context.DeadlineExceeded):
		log.Warn().Str("outcome", "deadline_exceeded").Msg(message)
	default:
		log.Error().Str("outcome", "error").Str("reason", "repository_error").Msg(message)
	}
}
