package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/repository/postgres"
)

func TestAccountRepository(t *testing.T) {
	pool, _ := sessionTestDatabase(t)
	ctx := context.Background()
	repo := postgres.NewUserRepo(pool)
	sessions := postgres.NewSessionRepo(pool)
	owner := sessionTestUser(t, pool)
	other := sessionTestUser(t, pool)
	const oldHash = "integration-test-hash"
	if _, err := repo.ChangeEmail(ctx, owner.ID, "stale-hash", "new@example.test"); !errors.Is(err, models.ErrCurrentPassword) {
		t.Fatalf("stale hash: %v", err)
	}
	if _, err := repo.ChangeEmail(ctx, owner.ID, oldHash, other.Email); !errors.Is(err, models.ErrEmailTaken) {
		t.Fatalf("duplicate email: %v", err)
	}
	updated, err := repo.ChangeEmail(ctx, owner.ID, oldHash, "new@example.test")
	if err != nil || updated.Email != "new@example.test" || updated.DisplayName != owner.DisplayName {
		t.Fatalf("email update: %+v %v", updated, err)
	}
	for i := 0; i < 2; i++ {
		if err := sessions.Create(ctx, sessionTestValue(owner.ID, time.Now().Add(time.Hour))); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.ChangePassword(ctx, owner.ID, oldHash, "next-hash"); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM auth_session WHERE account_id=$1`, string(owner.ID)).Scan(&count); err != nil || count != 0 {
		t.Fatalf("sessions count=%d err=%v", count, err)
	}
	got, err := repo.GetByID(ctx, owner.ID)
	if err != nil || got.PasswordHash != "next-hash" {
		t.Fatalf("hash not changed: %v", err)
	}
	if err := repo.DeleteAccount(ctx, owner.ID, oldHash); !errors.Is(err, models.ErrCurrentPassword) {
		t.Fatalf("stale delete: %v", err)
	}
	var trackID string
	if err := pool.QueryRow(ctx, `WITH audio AS (
	 INSERT INTO media_file(storage_key,mime_type,byte_size,duration_ms)
	 VALUES ('audio/account-test.wav','audio/wav',100,1000) RETURNING id)
	 INSERT INTO track(uploader_id,title,audio_file_id,status)
	 SELECT $1,'Shared track',id,'published' FROM audio RETURNING id::text`, string(owner.ID)).Scan(&trackID); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{
		`INSERT INTO playlist (owner_id,title) VALUES ($1,'Test playlist')`,
		`INSERT INTO listening_event (id,account_id,track_id,started_at) SELECT gen_random_uuid(),$1,t.id,CURRENT_TIMESTAMP FROM track t LIMIT 1`,
		`INSERT INTO favorite_track (account_id,track_id) SELECT $1,t.id FROM track t LIMIT 1`,
	} {
		if _, err := pool.Exec(ctx, query, string(owner.ID)); err != nil {
			t.Fatal(err)
		}
	}
	if err := sessions.Create(ctx, sessionTestValue(owner.ID, time.Now().Add(time.Hour))); err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteAccount(ctx, owner.ID, "next-hash"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetByID(ctx, owner.ID); !errors.Is(err, models.ErrUserNotFound) {
		t.Fatalf("deleted user: %v", err)
	}
	if _, err := repo.GetByID(ctx, other.ID); err != nil {
		t.Fatalf("other user changed: %v", err)
	}
	var uploader string
	if err := pool.QueryRow(ctx, `SELECT uploader_id::text FROM track WHERE id=$1`, trackID).Scan(&uploader); err != nil || uploader != "10000000-0000-4000-8000-000000000001" {
		t.Fatalf("shared track was not retained: uploader=%s err=%v", uploader, err)
	}
	for _, table := range []string{"auth_session", "playlist", "listening_event", "favorite_track"} {
		column := "account_id"
		if table == "playlist" {
			column = "owner_id"
		}
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE "+column+"=$1", string(owner.ID)).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s count=%d err=%v", table, count, err)
		}
	}
}
