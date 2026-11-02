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

var _ repository.UserRepositoryInterface = (*UserRepo)(nil)
var _ repository.ProfileRepositoryInterface = (*UserRepo)(nil)

// NewUserRepo создаёт репозиторий поверх уже открытого пула
func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

// Create вставляет аккаунт. id и created_at проставляет база.
// Повтор email — ErrEmailTaken.
func (r *UserRepo) Create(ctx context.Context, user models.User) (models.User, error) {
	const query = `
		INSERT INTO account (email, password_hash, display_name, avatar_file_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`

	var id string

	err := r.pool.QueryRow(ctx, query,
		user.Email, user.PasswordHash, user.DisplayName, mediaFileIDValue(user.AvatarFileID),
	).
		Scan(&id, &user.CreatedAt)
	if err != nil {
		if isEmailConflict(err) {
			return models.User{}, repository.ErrEmailTaken
		}
		return models.User{}, fmt.Errorf("insert account: %w", err)
	}

	user.ID = models.UserID(id)
	return user, nil
}

// GetByID возвращает пользователя или ErrUserNotFound
func (r *UserRepo) GetByID(ctx context.Context, id models.UserID) (models.User, error) {
	const query = `
		SELECT id, email, password_hash, display_name, avatar_file_id::text, created_at
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
		SELECT id, email, password_hash, display_name, avatar_file_id::text, created_at
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

// scanUser читает строку таблицы account в models.User.
func scanUser(row pgx.Row) (models.User, error) {
	var (
		user         models.User
		id           string
		avatarFileID *string
	)
	err := row.Scan(&id, &user.Email, &user.PasswordHash, &user.DisplayName, &avatarFileID, &user.CreatedAt)
	if err != nil {
		return models.User{}, err
	}

	user.ID = models.UserID(id)
	if avatarFileID != nil {
		fileID := models.MediaFileID(*avatarFileID)
		user.AvatarFileID = &fileID
	}
	return user, nil
}

// UpdateProfile changes the email, display name, and optional avatar atomically.
// A nil avatarFileID clears the current avatar.
func (r *UserRepo) UpdateProfile(
	ctx context.Context,
	id models.UserID,
	email, displayName string,
	avatarFileID *models.MediaFileID,
) (models.User, error) {
	const query = `
		UPDATE account
		SET email = $2, display_name = $3, avatar_file_id = $4
		WHERE id = $1::uuid
		RETURNING id::text, email, password_hash, display_name, avatar_file_id::text, created_at`

	var avatar any
	if avatarFileID != nil {
		avatar = string(*avatarFileID)
	}
	user, err := scanUser(r.pool.QueryRow(ctx, query, string(id), email, displayName, avatar))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidID(err) {
			return models.User{}, repository.ErrUserNotFound
		}
		if isEmailConflict(err) {
			return models.User{}, repository.ErrEmailTaken
		}
		return models.User{}, fmt.Errorf("update account profile: %w", err)
	}
	return user, nil
}

// isEmailConflict отличает занятый email от других нарушений уникальности.
func isEmailConflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "account_email_uk"
}

// проверка на неверный ID
func isInvalidID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}
