package repository

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"2026_2_Na_Vse_200/auth-microservice/sessions"
	"2026_2_Na_Vse_200/pkg/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const createSessionQuery = `
	INSERT INTO auth_session (id, account_id, expires_at, credential_version)
	SELECT $1::uuid, id, $3::timestamptz, credential_version
	FROM account
	WHERE id = $2 AND credential_version = $4
	FOR SHARE`

const getSessionQuery = `
	SELECT session.id::text, session.account_id::text, session.expires_at,
		session.credential_version, session.expires_at <= statement_timestamp() AS expired,
		session.credential_version <> account.credential_version AS revoked
	FROM auth_session AS session
	JOIN account ON account.id = session.account_id
	WHERE session.id = $1`

const deleteSessionQuery = `
	DELETE FROM auth_session
	WHERE id = $1`

const deleteInactiveSessionsQuery = `
	DELETE FROM auth_session AS session
	USING account
	WHERE account.id = session.account_id
	  AND (session.expires_at <= statement_timestamp() OR
	       session.credential_version <> account.credential_version)`

// SessionRepo хранит сессии в PostgreSQL.
type SessionRepo struct {
	pool *pgxpool.Pool
}

var _ sessions.Repository = (*SessionRepo)(nil)

// NewSessionRepo создаёт репозиторий сессий.
func NewSessionRepo(pool *pgxpool.Pool) *SessionRepo {
	return &SessionRepo{pool: pool}
}

// Create сохраняет сессию с идентификатором, заданным вызывающим кодом.
func (r *SessionRepo) Create(ctx context.Context, session sessions.Session) error {
	ctx = postgres.WithDBOperation(ctx, "sessions.create")
	// FOR SHARE synchronizes issuance with password changes on the account row.
	tag, err := r.pool.Exec(ctx, createSessionQuery,
		string(session.ID), string(session.UserID), session.ExpiresAt, session.CredentialVersion)
	if err != nil {
		return fmt.Errorf("insert auth session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return sessions.ErrCredentialsChanged
	}
	return nil
}

// GetByID проверяет срок действия сессии и версию учётных данных.
func (r *SessionRepo) GetByID(ctx context.Context, id sessions.SessionID) (sessions.Session, error) {
	ctx = postgres.WithDBOperation(ctx, "sessions.get_by_id")
	if err := ctx.Err(); err != nil {
		return sessions.Session{}, err
	}

	sessionID, err := uuid.Parse(string(id))
	if err != nil {
		return sessions.Session{}, sessions.ErrSessionNotFound
	}

	var (
		session           sessions.Session
		storedID, account string
		expired, revoked  bool
	)
	err = r.pool.QueryRow(ctx, getSessionQuery, sessionID.String()).
		Scan(&storedID, &account, &session.ExpiresAt, &session.CredentialVersion, &expired, &revoked)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sessions.Session{}, sessions.ErrSessionNotFound
		}
		return sessions.Session{}, fmt.Errorf("select auth session by id: %w", err)
	}
	if expired {
		return sessions.Session{}, sessions.ErrSessionExpired
	}
	if revoked {
		return sessions.Session{}, sessions.ErrSessionRevoked
	}

	session.ID = sessions.SessionID(storedID)
	session.UserID = sessions.UserID(account)
	return session, nil
}

// Delete удаляет сессию. Отсутствующий или некорректный ID не считается ошибкой.
func (r *SessionRepo) Delete(ctx context.Context, id sessions.SessionID) error {
	ctx = postgres.WithDBOperation(ctx, "sessions.delete")
	if err := ctx.Err(); err != nil {
		return err
	}

	sessionID, err := uuid.Parse(string(id))
	if err != nil {
		return nil
	}

	if err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, deleteSessionQuery, sessionID.String())
		return err
	}); err != nil {
		return fmt.Errorf("delete auth session: %w", err)
	}
	return nil
}

// DeleteInactive удаляет истёкшие и отозванные сессии.
func (r *SessionRepo) DeleteInactive(ctx context.Context) error {
	ctx = postgres.WithDBOperation(ctx, "sessions.delete_inactive")
	if err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, deleteInactiveSessionsQuery)
		return err
	}); err != nil {
		return fmt.Errorf("delete inactive auth sessions: %w", err)
	}
	return nil
}
