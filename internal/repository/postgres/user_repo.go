// репозиторий пользователей в postgres
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository"
)

// UserRepo хранит аккаунты в таблице account
type UserRepo struct {
	pool *pgxpool.Pool
}

// NewUserRepo создаёт репозиторий поверх уже открытого пула
func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

// Create вставляет аккаунт. id и created_at проставляет база.
// Повтор email — ErrEmailTaken.
func (r *UserRepo) Create(ctx context.Context, user models.User) (models.User, error) {
	const query = `
		INSERT INTO account (email, password_hash, display_name)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`

	var id string

	err := r.pool.QueryRow(ctx, query, user.Email, user.PasswordHash, user.DisplayName).
		Scan(&id, &user.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return models.User{}, repository.ErrEmailTaken
		}
		return models.User{}, fmt.Errorf("insert account: %w", err)
	}

	user.ID = models.ID(id)
	return user, nil
}

// GetByID возвращает пользователя или ErrUserNotFound
func (r *UserRepo) GetByID(ctx context.Context, id models.ID) (models.User, error) {
	const query = `
		SELECT id, email, password_hash, display_name, created_at
		FROM account
		WHERE id = $1::uuid`

	user, err := scanUser(r.pool.QueryRow(ctx, query, string(id)))
	if err != nil {
		if isInvalidID(err) || errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, repository.ErrUserNotFound
		}
		return models.User{}, fmt.Errorf("select account by id: %w", err)
	}
	return user, nil
}

// GetByEmail возвращает пользователя вместе с хешем пароля или ErrUserNotFound.
// Email ожидается в нижнем регистре.
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (models.User, error) {
	const query = `
		SELECT id, email, password_hash, display_name, created_at
		FROM account
		WHERE email = $1`

	user, err := scanUser(r.pool.QueryRow(ctx, query, email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, repository.ErrUserNotFound
		}
		return models.User{}, fmt.Errorf("select account by email: %w", err)
	}
	return user, nil
}

// 
func scanUser(row pgx.Row) (models.User, error) {
	var (
		user models.User
		id   string
	)
	err := row.Scan(&id, &user.Email, &user.PasswordHash, &user.DisplayName, &user.CreatedAt)
	if err != nil {
		return models.User{}, err
	}

	user.ID = models.ID(id)
	return user, nil
}

// проверка уникальности
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// проверка на неверный ID
func isInvalidID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}
