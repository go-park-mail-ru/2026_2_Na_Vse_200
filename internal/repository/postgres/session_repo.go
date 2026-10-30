package postgres

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const createSessionQuery = `
	INSERT INTO auth_session (id, account_id, expires_at)
	VALUES ($1, $2, $3)`

const getSessionQuery = `
	SELECT id::text, account_id::text, expires_at,
		expires_at <= statement_timestamp() AS expired
	FROM auth_session
	WHERE id = $1`

const deleteSessionQuery = `
	DELETE FROM auth_session
	WHERE id = $1`

const deleteExpiredSessionsQuery = `
	DELETE FROM auth_session
	WHERE expires_at <= statement_timestamp()`

// SessionRepo хранит серверные сессии в PostgreSQL.
type SessionRepo struct {
	pool *pgxpool.Pool
}

var _ ports.SessionRepository = (*SessionRepo)(nil)

// NewSessionRepo создаёт репозиторий поверх уже открытого пула.
func NewSessionRepo(pool *pgxpool.Pool) *SessionRepo {
	return &SessionRepo{pool: pool}
}

// Create сохраняет новую сессию. Идентификатор должен быть создан вызывающим кодом.
// Конфликт идентификатора и другие ошибки БД возвращаются вызывающему коду.
func (r *SessionRepo) Create(ctx context.Context, session models.Session) error {
	if _, err := r.pool.Exec(ctx, createSessionQuery, string(session.ID), string(session.UserID), session.ExpiresAt); err != nil {
		return fmt.Errorf("insert auth session: %w", err)
	}
	return nil
}

// GetByID возвращает действующую сессию. Время истечения проверяет база данных.
func (r *SessionRepo) GetByID(ctx context.Context, id models.SessionID) (models.Session, error) {
	if err := ctx.Err(); err != nil {
		return models.Session{}, err
	}

	sessionID, err := uuid.Parse(string(id))
	if err != nil {
		return models.Session{}, models.ErrSessionNotFound
	}

	var (
		session           models.Session
		storedID, account string
		expired           bool
	)
	err = r.pool.QueryRow(ctx, getSessionQuery, sessionID.String()).
		Scan(&storedID, &account, &session.ExpiresAt, &expired)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Session{}, models.ErrSessionNotFound
		}
		return models.Session{}, fmt.Errorf("select auth session by id: %w", err)
	}
	if expired {
		return models.Session{}, models.ErrSessionExpired
	}

	session.ID = models.SessionID(storedID)
	session.UserID = models.UserID(account)
	return session, nil
}

// Delete удаляет сессию. Отсутствующий или некорректный идентификатор допустим.
func (r *SessionRepo) Delete(ctx context.Context, id models.SessionID) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	sessionID, err := uuid.Parse(string(id))
	if err != nil {
		return nil
	}

	if _, err := r.pool.Exec(ctx, deleteSessionQuery, sessionID.String()); err != nil {
		return fmt.Errorf("delete auth session: %w", err)
	}
	return nil
}

// DeleteExpired удаляет все истёкшие сессии одним SQL-оператором.
func (r *SessionRepo) DeleteExpired(ctx context.Context) error {
	if _, err := r.pool.Exec(ctx, deleteExpiredSessionsQuery); err != nil {
		return fmt.Errorf("delete expired auth sessions: %w", err)
	}
	return nil
}
