package postgres

import (
	"context"
	"errors"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/pkg/postgres"
	"github.com/jackc/pgx/v5"
)

// changeAccount блокирует аккаунт и проверяет, что хеш пароля не изменился после проверки bcrypt.
func (r *UserRepo) changeAccount(ctx context.Context, id models.UserID, expectedHash string, change func(pgx.Tx) error) error {
	tx, err := r.transactions.Begin(ctx)
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
	ctx = postgres.WithDBOperation(ctx, "account.change_email")
	var user models.User
	err := r.changeAccount(ctx, id, expectedHash, func(tx pgx.Tx) error {
		var err error
		user, err = scanUser(tx.QueryRow(ctx, `UPDATE account SET email=$2 WHERE id=$1::uuid
		 RETURNING id::text,email,password_hash,display_name,avatar_file_id::text,created_at,credential_version`, string(id), email))
		if isEmailConflict(err) {
			return models.ErrEmailTaken
		}
		return err
	})
	return user, err
}

func (r *UserRepo) ChangePassword(ctx context.Context, id models.UserID, expectedHash, nextHash string) error {
	ctx = postgres.WithDBOperation(ctx, "account.change_password")
	return r.changeAccount(ctx, id, expectedHash, func(tx pgx.Tx) error {
		const query = `
			UPDATE account
			SET password_hash=$2, credential_version=credential_version+1
			WHERE id=$1::uuid`
		_, err := tx.Exec(ctx, query, string(id), nextHash)
		return err
	})
}

func (r *UserRepo) DeleteAccount(ctx context.Context, id models.UserID, expectedHash string) error {
	ctx = postgres.WithDBOperation(ctx, "account.delete")
	return r.changeAccount(ctx, id, expectedHash, func(tx pgx.Tx) error {
		var catalogOwner string
		if err := tx.QueryRow(ctx, "SELECT id::text FROM account WHERE system_key=$1 FOR SHARE", "catalog_owner").Scan(&catalogOwner); err != nil {
			return err
		}
		if catalogOwner == string(id) {
			return models.ErrForbidden
		}
		// Сохраняем треки за служебным аккаунтом; плейлисты, избранное и сессии удаляются каскадно.
		if _, err := tx.Exec(ctx, `UPDATE track SET uploader_id=$2::uuid WHERE uploader_id=$1::uuid`, string(id), catalogOwner); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM listening_event WHERE account_id=$1::uuid`, string(id)); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `DELETE FROM account WHERE id=$1::uuid`, string(id))
		return err
	})
}
