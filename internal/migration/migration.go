package migration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func newMigrator(ctx context.Context, db *sql.DB, files fs.FS) (*migrate.Migrate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	source, err := iofs.New(files, ".")
	if err != nil {
		return nil, fmt.Errorf("open SQL migrations: %w", err)
	}
	driver, err := migratepgx.WithInstance(db, &migratepgx.Config{})
	if err != nil {
		return nil, errors.Join(fmt.Errorf("initialize migration driver: %w", err), source.Close())
	}
	transactional := &transactionDriver{Driver: driver, db: db, ctx: ctx}
	runner, err := migrate.NewWithInstance("iofs", source, "pgx5", transactional)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("initialize migrations: %w", err), source.Close(), driver.Close())
	}
	runner.LockTimeout = lockTimeout
	return runner, nil
}

func applyMigrations(ctx context.Context, runner *migrate.Migrate, cmd command) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var err error
	switch cmd.action {
	case "up":
		err = runner.Up()
	case "down":
		err = runner.Steps(-cmd.steps)
	case "version":
		return nil
	default:
		return fmt.Errorf("unknown command %q", cmd.action)
	}
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("%s: %w", cmd.action, err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if errors.Is(err, migrate.ErrNoChange) {
		fmt.Println("no change")
	}
	return nil
}

func printVersion(runner *migrate.Migrate) error {
	version, dirty, err := runner.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		fmt.Println("version=none dirty=false")
		return nil
	}
	if err != nil {
		return fmt.Errorf("read migration version: %w", err)
	}
	fmt.Printf("version=%d dirty=%t\n", version, dirty)
	return nil
}
