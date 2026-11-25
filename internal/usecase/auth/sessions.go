package auth

import (
	"context"
	"time"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
)

type Sessions interface {
	CreateSession(context.Context, models.UserID) (models.Session, error)
	GetSession(context.Context, models.SessionID) (models.Session, error)
	DeleteSession(context.Context, models.SessionID) error
}

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
		return models.Session{}, err
	}
	return session, nil
}

func (s *SessionManager) GetSession(ctx context.Context, id models.SessionID) (models.Session, error) {
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
	if id == "" {
		return nil
	}
	return s.repo.Delete(ctx, id)
}

func (s *SessionManager) CheckProfileAccess(ctx context.Context, id models.SessionID, target models.UserID) error {
	session, err := s.GetSession(ctx, id)
	if err != nil {
		return err
	}
	if session.UserID != target {
		return models.ErrForbidden
	}
	return nil
}
