package postgres

import (
	"context"
	"errors"
	"fmt"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepo хранит пользователей в таблице account.
type UserRepo struct {
	pool *pgxpool.Pool
}

var _ ports.UserRepository = (*UserRepo)(nil)

// NewUserRepo создаёт репозиторий пользователей.
func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

// Create сохраняет аккаунт. ID и время создания задаёт база данных.
func (r *UserRepo) Create(ctx context.Context, user models.User) (models.User, error) {
	const query = `
		INSERT INTO account (email, password_hash, display_name, avatar_file_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`

	var id string
	var avatar any
	if user.AvatarFileID != nil {
		avatar = string(*user.AvatarFileID)
	}

	err := r.pool.QueryRow(ctx, query,
		user.Email, user.PasswordHash, user.DisplayName, avatar,
	).
		Scan(&id, &user.CreatedAt)
	if err != nil {
		if isEmailConflict(err) {
			return models.User{}, models.ErrEmailTaken
		}
		return models.User{}, fmt.Errorf("insert account: %w", err)
	}

	user.ID = models.UserID(id)
	return user, nil
}

// GetByID возвращает пользователя или models.ErrUserNotFound.
func (r *UserRepo) GetByID(ctx context.Context, id models.UserID) (models.User, error) {
	const query = `
		SELECT id, email, password_hash, display_name, avatar_file_id::text, created_at
		FROM account
		WHERE id = $1::uuid`

	user, err := scanUser(r.pool.QueryRow(ctx, query, string(id)))
	if err != nil {
		if isInvalidID(err) || errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, models.ErrUserNotFound
		}
		return models.User{}, fmt.Errorf("select account by id: %w", err)
	}
	return user, nil
}

// GetByEmail ищет пользователя по нормализованному email и возвращает его с хешем пароля.
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (models.User, error) {
	const query = `
		SELECT id, email, password_hash, display_name, avatar_file_id::text, created_at
		FROM account
		WHERE email = $1`

	user, err := scanUser(r.pool.QueryRow(ctx, query, email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, models.ErrUserNotFound
		}
		return models.User{}, fmt.Errorf("select account by email: %w", err)
	}
	return user, nil
}

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

func isEmailConflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "account_email_uk"
}

func isInvalidID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}
