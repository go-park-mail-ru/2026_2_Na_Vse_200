package migration

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"
	"uuid"

	"2026_2_Na_Vse_200/db/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func migrationDatabase(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set TEST_POSTGRES_DSN to run migrations in an isolated schema")
	}
	ctx := context.Background()
	admin, err := connectDB(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "migration_test_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.ExecContext(ctx, "CREATE SCHEMA "+quoted); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		defer admin.Close()
		if _, err := admin.ExecContext(context.Background(), "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Error(err)
		}
	})
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RuntimeParams == nil {
		cfg.RuntimeParams = make(map[string]string)
	}
	cfg.RuntimeParams["search_path"] = schema
	db := stdlib.OpenDB(*cfg)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestEmbeddedMigrationsRoundTrip(t *testing.T) {
	db := migrationDatabase(t)
	ctx := context.Background()
	runner, err := newMigrator(ctx, db, migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		a, b := runner.Close()
		if err := errors.Join(a, b); err != nil {
			t.Error(err)
		}
	})
	if err := applyMigrations(ctx, runner, command{action: "up"}); err != nil {
		t.Fatal(err)
	}
	version, dirty, err := runner.Version()
	if err != nil || version != 5 || dirty {
		t.Fatalf("version=%d dirty=%t error=%v", version, dirty, err)
	}
	if err := applyMigrations(ctx, runner, command{action: "up"}); err != nil {
		t.Fatal(err)
	}
	if err := applyMigrations(ctx, runner, command{action: "down", steps: 1}); err != nil {
		t.Fatal(err)
	}
	version, dirty, err = runner.Version()
	if err != nil || version != 4 || dirty {
		t.Fatalf("version=%d dirty=%t error=%v", version, dirty, err)
	}
	if err := applyMigrations(ctx, runner, command{action: "up"}); err != nil {
		t.Fatal(err)
	}
}

func TestFailedMigrationRollsBack(t *testing.T) {
	db := migrationDatabase(t)
	files := fstest.MapFS{
		"000001_first.up.sql":  {Data: []byte("BEGIN; CREATE TABLE kept (id int); COMMIT;")},
		"000002_second.up.sql": {Data: []byte("BEGIN; CREATE TABLE rolled_back (id int); SELECT 1/0; COMMIT;")},
	}
	ctx := context.Background()
	runner, err := newMigrator(ctx, db, files)
	if err != nil {
		t.Fatal(err)
	}
	defer runner.Close()
	if err := applyMigrations(ctx, runner, command{action: "up"}); err == nil {
		t.Fatal("invalid SQL succeeded")
	}
	assertTable(t, db, "kept", true)
	assertTable(t, db, "rolled_back", false)
	version, dirty, err := runner.Version()
	if err != nil || version != 2 || !dirty {
		t.Fatalf("version=%d dirty=%t error=%v", version, dirty, err)
	}
}

func TestCancelledMigrationRollsBack(t *testing.T) {
	db := migrationDatabase(t)
	files := fstest.MapFS{
		"000001_slow.up.sql": {Data: []byte("BEGIN; CREATE TABLE cancelled (id int); SELECT pg_sleep(10); COMMIT;")},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	runner, err := newMigrator(ctx, db, files)
	if err != nil {
		t.Fatal(err)
	}
	defer runner.Close()
	if err := applyMigrations(ctx, runner, command{action: "up"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	assertTable(t, db, "cancelled", false)
	version, dirty, err := runner.Version()
	if err != nil || version != 1 || !dirty {
		t.Fatalf("version=%d dirty=%t error=%v", version, dirty, err)
	}
}

func assertTable(t *testing.T, db *sql.DB, name string, want bool) {
	t.Helper()
	var table sql.NullString
	if err := db.QueryRowContext(context.Background(), "SELECT to_regclass($1)::text", name).Scan(&table); err != nil {
		t.Fatal(err)
	}
	if table.Valid != want {
		t.Fatalf("table %s exists=%t, want %t", name, table.Valid, want)
	}
}
