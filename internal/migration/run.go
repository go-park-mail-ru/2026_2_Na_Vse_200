package migration

import (
	"context"
	"errors"
	"fmt"
	"os"

	"2026_2_Na_Vse_200/db/migrations"
)

// Run выполняет команду миграций с подключением из POSTGRES_DSN.
func Run(ctx context.Context, args []string) (resultErr error) {
	cmd, err := parseCommand(args)
	if err != nil {
		return err
	}
	if cmd.action == "-h" || cmd.action == "--help" {
		fmt.Println(usage)
		return nil
	}
	db, err := connectDB(ctx, os.Getenv("POSTGRES_DSN"))
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, db.Close()) }()
	runner, err := newMigrator(ctx, db, migrations.Files)
	if err != nil {
		return err
	}
	defer func() {
		sourceErr, databaseErr := runner.Close()
		resultErr = errors.Join(resultErr, sourceErr, databaseErr)
	}()
	if err := applyMigrations(ctx, runner, cmd); err != nil {
		return err
	}
	return printVersion(runner)
}
