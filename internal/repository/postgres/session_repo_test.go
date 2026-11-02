package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/db/migrations"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository/postgres"
)

// TestSessionRepository runs only when explicitly given a PostgreSQL connection.
// Every run creates its own schema; existing schemas and their data are untouched.
func TestSessionRepository(t *testing.T) {
	pool, openPool := sessionTestDatabase(t)
	repo := postgres.NewSessionRepo(pool)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	t.Run("persists across independent pools", func(t *testing.T) {
		owner := sessionTestUser(t, pool)
		session := sessionTestValue(owner.ID, time.Now().Add(time.Hour))
		writerPool := openPool(t)
		if err := postgres.NewSessionRepo(writerPool).Create(ctx, session); err != nil {
			t.Fatalf("Create: %v", err)
		}
		writerPool.Close()

		readerPool := openPool(t)
		got, err := postgres.NewSessionRepo(readerPool).GetByID(ctx, session.ID)
		if err != nil {
			t.Fatalf("GetByID through a new pool: %v", err)
		}
		if got.ID != session.ID || got.UserID != session.UserID || !got.ExpiresAt.Equal(session.ExpiresAt) {
			t.Fatalf("stored session = %+v, want %+v", got, session)
		}
	})

	t.Run("unknown and malformed IDs", func(t *testing.T) {
		for _, id := range []models.SessionID{models.SessionID(uuid.New().String()), "invalid-session", ""} {
			got, err := repo.GetByID(ctx, id)
			if !errors.Is(err, repository.ErrSessionNotFound) {
				t.Errorf("GetByID(%q): %v, want ErrSessionNotFound", id, err)
			}
			if got != (models.Session{}) {
				t.Errorf("GetByID(%q) returned a session on failure: %+v", id, got)
			}
			if err := repo.Delete(ctx, id); err != nil {
				t.Errorf("Delete(%q) must be idempotent: %v", id, err)
			}
		}
	})

	t.Run("expired sessions cannot authenticate", func(t *testing.T) {
		owner := sessionTestUser(t, pool)
		session := sessionTestValue(owner.ID, time.Now().Add(-time.Minute))
		if err := repo.Create(ctx, session); err != nil {
			t.Fatalf("Create expired session: %v", err)
		}
		got, err := repo.GetByID(ctx, session.ID)
		if !errors.Is(err, repository.ErrSessionExpired) {
			t.Fatalf("GetByID expired: %v, want ErrSessionExpired", err)
		}
		if got != (models.Session{}) {
			t.Fatalf("expired session leaked to caller: %+v", got)
		}

		// Use the database clock, without sleeps or an application clock offset.
		const boundaryQuery = `
			UPDATE auth_session SET expires_at = statement_timestamp() WHERE id = $1::uuid`
		if _, err := pool.Exec(ctx, boundaryQuery, string(session.ID)); err != nil {
			t.Fatalf("set expiry to database time: %v", err)
		}
		if _, err := repo.GetByID(ctx, session.ID); !errors.Is(err, repository.ErrSessionExpired) {
			t.Fatalf("session whose deadline has been reached: %v, want ErrSessionExpired", err)
		}
	})

	t.Run("logout revokes the stored session", func(t *testing.T) {
		owner := sessionTestUser(t, pool)
		session := sessionTestValue(owner.ID, time.Now().Add(time.Hour))
		if err := repo.Create(ctx, session); err != nil {
			t.Fatalf("Create: %v", err)
		}
		if err := repo.Delete(ctx, session.ID); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if _, err := repo.GetByID(ctx, session.ID); !errors.Is(err, repository.ErrSessionNotFound) {
			t.Fatalf("GetByID after logout: %v, want ErrSessionNotFound", err)
		}
		if err := repo.Delete(ctx, session.ID); err != nil {
			t.Fatalf("second Delete: %v", err)
		}
	})

	t.Run("cleanup deletes only expired sessions", func(t *testing.T) {
		owner := sessionTestUser(t, pool)
		active := sessionTestValue(owner.ID, time.Now().Add(time.Hour))
		expired := sessionTestValue(owner.ID, time.Now().Add(-time.Hour))
		for _, session := range []models.Session{active, expired} {
			if err := repo.Create(ctx, session); err != nil {
				t.Fatalf("Create: %v", err)
			}
		}
		if err := repo.DeleteExpired(ctx); err != nil {
			t.Fatalf("DeleteExpired: %v", err)
		}
		if _, err := repo.GetByID(ctx, active.ID); err != nil {
			t.Fatalf("active session must survive cleanup: %v", err)
		}
		if _, err := repo.GetByID(ctx, expired.ID); !errors.Is(err, repository.ErrSessionNotFound) {
			t.Fatalf("cleaned session: %v, want ErrSessionNotFound", err)
		}
		if err := repo.DeleteExpired(ctx); err != nil {
			t.Fatalf("repeated cleanup: %v", err)
		}
	})

	t.Run("duplicate ID never overwrites a session", func(t *testing.T) {
		owner := sessionTestUser(t, pool)
		otherOwner := sessionTestUser(t, pool)
		original := sessionTestValue(owner.ID, time.Now().Add(time.Hour))
		if err := repo.Create(ctx, original); err != nil {
			t.Fatalf("Create: %v", err)
		}
		duplicate := original
		duplicate.UserID = otherOwner.ID
		duplicate.ExpiresAt = original.ExpiresAt.Add(time.Hour)
		var pgErr *pgconn.PgError
		if err := repo.Create(ctx, duplicate); !errors.As(err, &pgErr) || pgErr.Code != "23505" {
			t.Fatalf("duplicate Create: %v, want unique violation", err)
		}
		got, err := repo.GetByID(ctx, original.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.UserID != original.UserID || !got.ExpiresAt.Equal(original.ExpiresAt) {
			t.Fatalf("duplicate Create changed the original session: %+v", got)
		}
	})

	t.Run("account foreign key and cascading deletion", func(t *testing.T) {
		missing := sessionTestValue(models.UserID(uuid.New().String()), time.Now().Add(time.Hour))
		var pgErr *pgconn.PgError
		if err := repo.Create(ctx, missing); !errors.As(err, &pgErr) || pgErr.Code != "23503" {
			t.Fatalf("Create with unknown account: %v, want foreign key violation", err)
		}
		owner := sessionTestUser(t, pool)
		session := sessionTestValue(owner.ID, time.Now().Add(time.Hour))
		if err := repo.Create(ctx, session); err != nil {
			t.Fatalf("Create: %v", err)
		}
		const deleteAccountQuery = `DELETE FROM account WHERE id = $1::uuid`
		if _, err := pool.Exec(ctx, deleteAccountQuery, string(owner.ID)); err != nil {
			t.Fatalf("delete account: %v", err)
		}
		if _, err := repo.GetByID(ctx, session.ID); !errors.Is(err, repository.ErrSessionNotFound) {
			t.Fatalf("session after account deletion: %v, want ErrSessionNotFound", err)
		}
	})

	t.Run("canceled context reaches every database operation", func(t *testing.T) {
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		session := sessionTestValue(models.UserID(uuid.New().String()), time.Now().Add(time.Hour))
		checks := []struct {
			name string
			run  func() error
		}{
			{"Create", func() error { return repo.Create(canceled, session) }},
			{"GetByID", func() error { _, err := repo.GetByID(canceled, session.ID); return err }},
			{"Delete", func() error { return repo.Delete(canceled, session.ID) }},
			{"DeleteExpired", func() error { return repo.DeleteExpired(canceled) }},
		}
		for _, check := range checks {
			if err := check.run(); !errors.Is(err, context.Canceled) {
				t.Errorf("%s: %v, want context.Canceled", check.name, err)
			}
		}
	})

	t.Run("database failures are not authentication failures", func(t *testing.T) {
		closedPool := openPool(t)
		closedPool.Close()
		closedRepo := postgres.NewSessionRepo(closedPool)
		session := sessionTestValue(models.UserID(uuid.New().String()), time.Now().Add(time.Hour))
		checks := []struct {
			name string
			run  func() error
		}{
			{"Create", func() error { return closedRepo.Create(ctx, session) }},
			{"GetByID", func() error { _, err := closedRepo.GetByID(ctx, session.ID); return err }},
			{"Delete", func() error { return closedRepo.Delete(ctx, session.ID) }},
			{"DeleteExpired", func() error { return closedRepo.DeleteExpired(ctx) }},
		}
		for _, check := range checks {
			err := check.run()
			if err == nil || errors.Is(err, repository.ErrSessionNotFound) || errors.Is(err, repository.ErrSessionExpired) {
				t.Errorf("%s must preserve the database failure, got %v", check.name, err)
			}
		}
	})
}

func sessionTestDatabase(t *testing.T) (*pgxpool.Pool, func(*testing.T) *pgxpool.Pool) {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set TEST_POSTGRES_DSN to run PostgreSQL integration tests in an isolated schema")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect to test PostgreSQL: %v", err)
	}
	schema := "session_test_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		_ = admin.Close(ctx)
		t.Fatalf("create isolated test schema: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		defer admin.Close(cleanupCtx)
		// quotedSchema is generated above and never comes from the connection string.
		if _, err := admin.Exec(cleanupCtx, "DROP SCHEMA "+quotedSchema+" CASCADE"); err != nil {
			t.Errorf("drop isolated test schema %s: %v", schema, err)
		}
	})

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse TEST_POSTGRES_DSN: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	openPool := func(t *testing.T) *pgxpool.Pool {
		t.Helper()
		pool, err := pgxpool.NewWithConfig(context.Background(), cfg.Copy())
		if err != nil {
			t.Fatalf("open isolated test pool: %v", err)
		}
		t.Cleanup(pool.Close)
		return pool
	}
	pool := openPool(t)
	files, err := fs.Glob(migrations.Files, "*.up.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("find SQL migrations: %v, files=%v", err, files)
	}
	for _, file := range files {
		sql, err := migrations.Files.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		if _, err := pool.Exec(ctx, string(sql), pgx.QueryExecModeSimpleProtocol); err != nil {
			t.Fatalf("apply %s to isolated schema: %v", file, err)
		}
	}
	return pool, openPool
}

func sessionTestUser(t *testing.T, pool *pgxpool.Pool) models.User {
	t.Helper()
	user, err := postgres.NewUserRepo(pool).Create(context.Background(), models.User{
		Email:        fmt.Sprintf("session-%s@example.test", uuid.New().String()),
		PasswordHash: "integration-test-hash",
		DisplayName:  "Session integration test",
	})
	if err != nil {
		t.Fatalf("create test account: %v", err)
	}
	return user
}

func sessionTestValue(owner models.UserID, expiresAt time.Time) models.Session {
	return models.Session{
		ID:        models.SessionID(uuid.New().String()),
		UserID:    owner,
		ExpiresAt: expiresAt.UTC().Truncate(time.Microsecond),
	}
}
