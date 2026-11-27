package sessions

import (
	"context"
	"errors"
	"time"
)

type SessionID string
type UserID string

type Session struct {
	ID                SessionID
	UserID            UserID
	ExpiresAt         time.Time
	CredentialVersion int64
}

func (s Session) IsExpired(now time.Time) bool {
	return !now.Before(s.ExpiresAt)
}

var (
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionExpired     = errors.New("session expired")
	ErrSessionRevoked     = errors.New("session revoked")
	ErrCredentialsChanged = errors.New("credentials changed")
	ErrForbidden          = errors.New("profile access denied")
)

// Repository stores sessions for Auth.
type Repository interface {
	Create(ctx context.Context, session Session) error
	GetByID(ctx context.Context, id SessionID) (Session, error)
	// Delete treats a missing or invalid ID as a no-op.
	Delete(ctx context.Context, id SessionID) error
}
