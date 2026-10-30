package postgres

import (
	"context"
	"errors"
	"fmt"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
	"github.com/jackc/pgx/v5"
)

// Профиль использует тот же UserRepo и таблицу account.
var _ ports.ProfileRepository = (*UserRepo)(nil)

func (r *UserRepo) UpdateProfile(ctx context.Context, id models.UserID, email, displayName string) (models.User, error) {
	const query = `
 UPDATE account SET email = $2, display_name = $3
 WHERE id = $1::uuid
 RETURNING id::text, email, password_hash, display_name, avatar_file_id::text, created_at`
	user, err := scanUser(r.pool.QueryRow(ctx, query, string(id), email, displayName))
	if isEmailConflict(err) {
		return models.User{}, models.ErrEmailTaken
	}
	if errors.Is(err, pgx.ErrNoRows) || isInvalidID(err) {
		return models.User{}, models.ErrUserNotFound
	}
	if err != nil {
		return models.User{}, fmt.Errorf("update profile: %w", err)
	}
	return user, nil
}

// SetAvatar привязывает уже сохранённый файл к пользователю.
func (r *UserRepo) SetAvatar(ctx context.Context, id models.UserID, fileID models.MediaFileID) (models.User, error) {
	const query = `
 UPDATE account SET avatar_file_id = $2::uuid
 WHERE id = $1::uuid
 RETURNING id::text, email, password_hash, display_name, avatar_file_id::text, created_at`
	user, err := scanUser(r.pool.QueryRow(ctx, query, string(id), string(fileID)))
	if errors.Is(err, pgx.ErrNoRows) {
		return models.User{}, models.ErrUserNotFound
	}
	if err != nil {
		return models.User{}, fmt.Errorf("set avatar: %w", err)
	}
	return user, nil
}
