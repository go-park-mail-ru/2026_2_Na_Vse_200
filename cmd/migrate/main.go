// Команда migrate применяет SQL-миграции к базе из POSTGRES_DSN.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/db/migrations"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Printf("миграции: %v", err)
		os.Exit(1)
	}
}

func run(args []string) (resultErr error) {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Println("usage: migrate up | version | down <positive steps>")
		return nil
	}
	if len(args) == 0 {
		return errors.New("usage: migrate up | version | down <positive steps>")
	}

	action := args[0]
	steps := 0
	switch action {
	case "up", "version":
		if len(args) != 1 {
			return fmt.Errorf("%s does not accept arguments", action)
		}
	case "down":
		if len(args) != 2 {
			return errors.New("usage: migrate down <positive steps>")
		}
		var err error
		steps, err = strconv.Atoi(args[1])
		if err != nil || steps <= 0 {
			return errors.New("down requires a positive number of steps")
		}
	default:
		return fmt.Errorf("unknown command %q", action)
	}

	dsn := strings.TrimSpace(os.Getenv("POSTGRES_DSN"))
	if dsn == "" {
		return errors.New("POSTGRES_DSN is not set")
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		// Parse errors may contain the connection string, including its password.
		return errors.New("POSTGRES_DSN has invalid syntax")
	}
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = 5 * time.Second
	}

	// database/sql здесь использует только драйвер pgx/v5, необходимый адаптеру migrate.
	db := stdlib.OpenDB(*cfg)
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = db.PingContext(ctx)
	cancel()
	if err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	source, err := iofs.New(migrations.Files, ".")
	if err != nil {
		return fmt.Errorf("open SQL migrations: %w", err)
	}
	driver, err := migratepgx.WithInstance(db, &migratepgx.Config{
		StatementTimeout: time.Minute,
	})
	if err != nil {
		_ = source.Close()
		return fmt.Errorf("initialize pgx migration driver: %w", err)
	}
	runner, err := migrate.NewWithInstance("iofs", source, "pgx5", driver)
	if err != nil {
		_ = source.Close()
		_ = driver.Close()
		return fmt.Errorf("initialize migrations: %w", err)
	}
	defer func() {
		sourceErr, databaseErr := runner.Close()
		resultErr = errors.Join(resultErr, sourceErr, databaseErr)
	}()

	runner.LockTimeout = 15 * time.Second
	switch action {
	case "up":
		err = runner.Up()
	case "down":
		err = runner.Steps(-steps)
	}
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("%s: %w", action, err)
	}
	if errors.Is(err, migrate.ErrNoChange) {
		fmt.Println("no change")
	}

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
