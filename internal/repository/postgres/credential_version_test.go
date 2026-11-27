package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"2026_2_Na_Vse_200/auth-microservice/repository"
	"2026_2_Na_Vse_200/auth-microservice/sessions"
	"2026_2_Na_Vse_200/internal/repository/postgres"
)

func TestCredentialVersionMigration(t *testing.T) {
	const up = "000008_credential_version.up.sql"
	pool, _ := sessionTestDatabaseBefore(t, up)
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()

	var ownerID string
	err := pool.QueryRow(ctx, `
		INSERT INTO account (email, password_hash, display_name)
		VALUES ('legacy@example.test', 'old-hash', 'Legacy account')
		RETURNING id::text`).Scan(&ownerID)
	if err != nil {
		t.Fatal(err)
	}
	legacyID := sessions.SessionID(uuid.New().String())
	_, err = pool.Exec(ctx, `
		INSERT INTO auth_session (id, account_id, expires_at)
		VALUES ($1, $2, statement_timestamp() + interval '1 hour')`, string(legacyID), ownerID)
	if err != nil {
		t.Fatal(err)
	}

	applySessionTestMigration(t, ctx, pool, up)
	authRepo := repository.NewSessionRepo(pool)
	legacy, err := authRepo.GetByID(ctx, legacyID)
	if err != nil || legacy.CredentialVersion != 1 {
		t.Fatalf("legacy session after update: version=%d err=%v", legacy.CredentialVersion, err)
	}
	userRepo := postgres.NewUserRepo(pool)
	owner, err := userRepo.GetByEmail(ctx, "legacy@example.test")
	if err != nil || owner.CredentialVersion != 1 {
		t.Fatalf("legacy account after update: version=%d err=%v", owner.CredentialVersion, err)
	}
	if err := userRepo.ChangePassword(ctx, owner.ID, "old-hash", "new-hash"); err != nil {
		t.Fatal(err)
	}
	if _, err := authRepo.GetByID(ctx, legacyID); !errors.Is(err, sessions.ErrSessionRevoked) {
		t.Fatalf("legacy session after password change: %v, want revoked", err)
	}
	current := sessionTestValue(owner.ID, time.Now().Add(time.Hour))
	current.CredentialVersion = 2
	if err := authRepo.Create(ctx, current); err != nil {
		t.Fatal(err)
	}

	applySessionTestMigration(t, ctx, pool, "000008_credential_version.down.sql")
	var legacyExists, currentExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM auth_session WHERE id = $1),
		       EXISTS (SELECT 1 FROM auth_session WHERE id = $2)`,
		string(legacyID), string(current.ID)).Scan(&legacyExists, &currentExists)
	if err != nil || legacyExists || !currentExists {
		t.Fatalf("sessions after rollback: legacy=%v current=%v err=%v", legacyExists, currentExists, err)
	}
}

func TestSessionIssuanceDuringPasswordChange(t *testing.T) {
	pool, openPool := sessionTestDatabase(t)
	owner := sessionTestUser(t, pool)
	authRepo := repository.NewSessionRepo(openPool(t))
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()

	change, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer change.Rollback(t.Context())
	if _, err := change.Exec(ctx, `
		UPDATE account SET password_hash = 'new-hash', credential_version = credential_version + 1
		WHERE id = $1`, string(owner.ID)); err != nil {
		t.Fatal(err)
	}

	stale := sessionTestValue(owner.ID, time.Now().Add(time.Hour))
	result := make(chan error, 1)
	go func() { result <- authRepo.Create(ctx, stale) }()

	// The request must reach PostgreSQL while the password update is still open.
	blocker := int32(change.Conn().PgConn().PID())
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var blocked bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_stat_activity
				WHERE $1 = ANY(pg_blocking_pids(pid))
			)`, blocker).Scan(&blocked)
		if err != nil {
			t.Fatal(err)
		}
		if blocked {
			break
		}
		select {
		case err := <-result:
			t.Fatalf("session issuance did not wait for password update: %v", err)
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-ticker.C:
		}
	}
	if err := change.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, sessions.ErrCredentialsChanged) {
			t.Fatalf("issuance with stale version: %v, want ErrCredentialsChanged", err)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if _, err := authRepo.GetByID(ctx, stale.ID); !errors.Is(err, sessions.ErrSessionNotFound) {
		t.Fatalf("stale session was stored: %v", err)
	}
	stale.CredentialVersion = owner.CredentialVersion + 1
	if err := authRepo.Create(ctx, stale); err != nil {
		t.Fatalf("issuance with current version: %v", err)
	}
}
