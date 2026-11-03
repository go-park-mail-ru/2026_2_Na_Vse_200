package migration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/golang-migrate/migrate/v4/database"
)

// Библиотека управляет версиями и блокировкой, а Run — транзакцией одного SQL-файла.
type transactionDriver struct {
	database.Driver
	db  *sql.DB
	ctx context.Context
}

func (d *transactionDriver) Run(reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("read migration: %w", err)
	}
	query, err := migrationBody(string(data))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(d.ctx, migrationTimeout)
	defer cancel()
	return executeMigration(ctx, d.db, query)
}

func migrationBody(query string) (string, error) {
	query = strings.TrimSpace(query)
	begins := strings.HasPrefix(query, "BEGIN;")
	commits := strings.HasSuffix(query, "COMMIT;")
	if begins != commits {
		return "", errors.New("migration must have both BEGIN and COMMIT, or neither")
	}
	if begins {

		query = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(query, "BEGIN;"), "COMMIT;"))
	}
	return query, nil
}

func executeMigration(ctx context.Context, db *sql.DB, query string) (resultErr error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			resultErr = errors.Join(resultErr, fmt.Errorf("rollback migration: %w", err))
		}
	}()
	if _, err := tx.ExecContext(ctx, query); err != nil {
		return errors.Join(fmt.Errorf("execute migration: %w", err), ctx.Err())
	}
	if err := tx.Commit(); err != nil {
		return errors.Join(fmt.Errorf("commit migration: %w", err), ctx.Err())
	}
	return nil
}
