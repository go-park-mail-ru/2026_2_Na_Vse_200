package postgres

import (
	"context"
	"errors"

	"2026_2_Na_Vse_200/internal/models"
	"github.com/jackc/pgx/v5"
)

// Serialize sensitive changes and verify that the password did not change after bcrypt verification.
func (r *UserRepo) changeAccount(ctx context.Context, id models.UserID, expectedHash string, change func(pgx.Tx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var hash string
	err = tx.QueryRow(ctx, `SELECT password_hash FROM account WHERE id=$1::uuid FOR UPDATE`, string(id)).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.ErrUnauthorized
	}
	if err != nil {
		return err
	}
	if hash != expectedHash {
		return models.ErrCurrentPassword
	}
	if err = change(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *UserRepo) ChangeEmail(ctx context.Context, id models.UserID, expectedHash, email string) (models.User, error) {
	var user models.User
	err := r.changeAccount(ctx, id, expectedHash, func(tx pgx.Tx) error {
		var err error
		user, err = scanUser(tx.QueryRow(ctx, `UPDATE account SET email=$2 WHERE id=$1::uuid
		 RETURNING id::text,email,password_hash,display_name,avatar_file_id::text,created_at`, string(id), email))
		if isEmailConflict(err) {
			return models.ErrEmailTaken
		}
		return err
	})
	return user, err
}

func (r *UserRepo) ChangePassword(ctx context.Context, id models.UserID, expectedHash, nextHash string) error {
	return r.changeAccount(ctx, id, expectedHash, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE account SET password_hash=$2 WHERE id=$1::uuid`, string(id), nextHash); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `DELETE FROM auth_session WHERE account_id=$1::uuid`, string(id))
		return err
	})
}

func (r *UserRepo) DeleteAccount(ctx context.Context, id models.UserID, expectedHash string) error {
	if string(id) == "10000000-0000-4000-8000-000000000001" {
		return models.ErrForbidden
	}
	return r.changeAccount(ctx, id, expectedHash, func(tx pgx.Tx) error {
		// Retain the shared music catalog; user-owned playlists, favorites and sessions cascade.
		if _, err := tx.Exec(ctx, `UPDATE track SET uploader_id='10000000-0000-4000-8000-000000000001'::uuid WHERE uploader_id=$1::uuid`, string(id)); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM listening_event WHERE account_id=$1::uuid`, string(id)); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `DELETE FROM account WHERE id=$1::uuid`, string(id))
		return err
	})
}
