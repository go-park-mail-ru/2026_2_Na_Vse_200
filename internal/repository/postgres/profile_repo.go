package postgres

import (
	"context"
	"errors"
	"fmt"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
	"2026_2_Na_Vse_200/pkg/postgres"
	"github.com/jackc/pgx/v5"
)

var _ ports.ProfileRepository = (*UserRepo)(nil)

func (r *UserRepo) UpdateProfile(ctx context.Context, id models.UserID, email, displayName string) (models.User, error) {
	ctx = postgres.WithDBOperation(ctx, "profile.update")
	const query = `
 UPDATE account SET display_name = $3
 WHERE id = $1::uuid AND email = $2
 RETURNING id::text, email, password_hash, display_name, avatar_file_id::text, created_at, credential_version`
	var user models.User
	err := pgx.BeginFunc(ctx, r.transactions, func(tx pgx.Tx) error {
		var err error
		user, err = scanUser(tx.QueryRow(ctx, query, string(id), email, displayName))
		return err
	})
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
	ctx = postgres.WithDBOperation(ctx, "profile.set_avatar")
	var user models.User
	err := pgx.BeginFunc(ctx, r.transactions, func(tx pgx.Tx) error {
		var err error
		user, err = setAvatar(ctx, tx, id, fileID)
		return err
	})
	if err != nil {
		return models.User{}, fmt.Errorf("set avatar: %w", err)
	}
	return user, nil
}

// SaveAvatar создаёт метаданные и привязывает аватар в одной транзакции.
func (r *UserRepo) SaveAvatar(ctx context.Context, id models.UserID, file models.MediaFile) (models.User, error) {
	ctx = postgres.WithDBOperation(ctx, "profile.save_avatar")
	var user models.User
	err := pgx.BeginFunc(ctx, r.transactions, func(tx pgx.Tx) error {
		saved, err := createMedia(ctx, tx, file)
		if err != nil {
			return err
		}
		user, err = setAvatar(ctx, tx, id, saved.ID)
		return err
	})
	if err != nil {
		return models.User{}, fmt.Errorf("save avatar: %w", err)
	}
	return user, nil
}

func setAvatar(ctx context.Context, db rowQuerier, id models.UserID, fileID models.MediaFileID) (models.User, error) {
	const query = `
 UPDATE account SET avatar_file_id = $2::uuid
 WHERE id = $1::uuid
 RETURNING id::text, email, password_hash, display_name, avatar_file_id::text, created_at, credential_version`
	user, err := scanUser(db.QueryRow(ctx, query, string(id), string(fileID)))
	if errors.Is(err, pgx.ErrNoRows) || isInvalidID(err) {
		return models.User{}, models.ErrUserNotFound
	}
	if err != nil {
		return models.User{}, fmt.Errorf("set avatar: %w", err)
	}
	return user, nil
}
