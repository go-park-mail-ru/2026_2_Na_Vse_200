package migrations_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"2026_2_Na_Vse_200/db/migrations"
)

func TestLocalStorageUpgrade(t *testing.T) {
	conn, ctx := localStorageDatabase(t)
	audio := legacyMedia(t, conn, ctx, "audio", "shared/song.ogg", false)
	other := legacyMedia(t, conn, ctx, "another-audio", "shared/song.ogg", false)
	cover := legacyMedia(t, conn, ctx, "covers", "обложка.webp", true)
	const links = `INSERT INTO track (uploader_id, title, audio_file_id, cover_file_id)
		VALUES ('10000000-0000-4000-8000-000000000001', 'Test track', $1::uuid, $2::uuid)`
	localStorageExec(t, conn, ctx, links, audio, cover)
	before := mediaMetadata(t, conn, ctx)
	if err := localStorageMigration(t, conn, ctx, "000005_media_local_storage.up.sql"); err != nil {
		t.Fatal(err)
	}
	assertMediaColumns(t, conn, ctx, true)
	if got := mediaMetadata(t, conn, ctx); got != before {
		t.Fatalf("upgrade changed media metadata: before=%s after=%s", before, got)
	}
	for id, key := range map[string]string{audio: "audio/shared/song.ogg", other: "another-audio/shared/song.ogg", cover: "covers/обложка.webp"} {
		var got string
		if err := conn.QueryRow(ctx, `SELECT storage_key FROM media_file WHERE id = $1::uuid`, id).Scan(&got); err != nil || got != key {
			t.Fatalf("storage key for %s: got %q, error=%v, want %q", id, got, err, key)
		}
	}
	var linkedAudio, linkedCover string
	if err := conn.QueryRow(ctx, `SELECT audio_file_id::text, cover_file_id::text FROM track`).Scan(&linkedAudio, &linkedCover); err != nil || linkedAudio != audio || linkedCover != cover {
		t.Fatalf("media foreign keys changed: audio=%s cover=%s error=%v", linkedAudio, linkedCover, err)
	}

	const insertKey = `INSERT INTO media_file (storage_key, mime_type, byte_size) VALUES ($1, 'image/webp', 10)`
	for _, key := range []string{"standalone.webp", "Музыка/Песня 01.webp", "covers/cover image.webp", "a/.hidden", " space / file name "} {
		localStorageExec(t, conn, ctx, insertKey, key)
	}
	_, err := conn.Exec(ctx, insertKey, "covers/cover image.webp")
	localStorageSQLState(t, err, "23505")
	for _, key := range []string{"", "   ", "/absolute", "trailing/", "a//b", ".", "..", "a/./b", "a/../b", `a\b`, "C:/file", "a:b", "a/line\nb", "a/tab\tb", "a/\x01b", "a/\x7fb"} {
		t.Run(fmt.Sprintf("reject %q", key), func(t *testing.T) {
			_, err := conn.Exec(ctx, insertKey, key)
			localStorageSQLState(t, err, "23514")
		})
	}
}

func TestLocalStorageLegacyRoundTrip(t *testing.T) {
	conn, ctx := localStorageDatabase(t)
	legacyMedia(t, conn, ctx, " Музыка ", " Исполнитель/Песня 01.ogg ", false)
	legacyMedia(t, conn, ctx, "covers", "same.webp", true)
	legacyMedia(t, conn, ctx, "other-covers", "same.webp", true)
	before := legacyMediaLocations(t, conn, ctx)
	metadata := mediaMetadata(t, conn, ctx)
	for _, file := range []string{"000005_media_local_storage.up.sql", "000005_media_local_storage.down.sql"} {
		if err := localStorageMigration(t, conn, ctx, file); err != nil {
			t.Fatal(err)
		}
	}
	assertMediaColumns(t, conn, ctx, false)
	if got := legacyMediaLocations(t, conn, ctx); got != before {
		t.Fatalf("legacy locations did not round-trip exactly: before=%s after=%s", before, got)
	}
	if err := localStorageMigration(t, conn, ctx, "000005_media_local_storage.up.sql"); err != nil {
		t.Fatal(err)
	}
	if got := mediaMetadata(t, conn, ctx); got != metadata {
		t.Fatalf("round-trip changed media metadata: before=%s after=%s", metadata, got)
	}
}

func TestLocalStorageInvalidLegacyRollback(t *testing.T) {
	conn, ctx := localStorageDatabase(t)
	legacyMedia(t, conn, ctx, "good", "song.ogg", false)
	for _, invalid := range []struct{ bucket, key string }{
		{"nested/bucket", "song.ogg"}, {"audio", "../song.ogg"}, {"audio", "/song.ogg"}, {"audio", `folder\song.ogg`}, {"audio", "part//song.ogg"},
	} {
		id := legacyMedia(t, conn, ctx, invalid.bucket, invalid.key, false)
		before := legacyMediaLocations(t, conn, ctx)
		metadata := mediaMetadata(t, conn, ctx)
		if err := localStorageMigration(t, conn, ctx, "000005_media_local_storage.up.sql"); err == nil {
			t.Fatalf("upgrade accepted invalid legacy location %q / %q", invalid.bucket, invalid.key)
		}
		assertMediaColumns(t, conn, ctx, false)
		if got := legacyMediaLocations(t, conn, ctx); got != before {
			t.Fatalf("failed upgrade changed original locations: before=%s after=%s", before, got)
		}
		if got := mediaMetadata(t, conn, ctx); got != metadata {
			t.Fatal("failed upgrade changed media metadata")
		}
		localStorageExec(t, conn, ctx, `DELETE FROM media_file WHERE id = $1::uuid`, id)
	}
	if err := localStorageMigration(t, conn, ctx, "000005_media_local_storage.up.sql"); err != nil {
		t.Fatalf("upgrade after repairing invalid legacy data: %v", err)
	}
}

func TestLocalStorageGuardedDowngradeRollback(t *testing.T) {
	conn, ctx := localStorageDatabase(t)
	legacyMedia(t, conn, ctx, "original", "image.webp", true)
	if err := localStorageMigration(t, conn, ctx, "000005_media_local_storage.up.sql"); err != nil {
		t.Fatal(err)
	}
	for i, key := range []string{"single-component.webp", " /image.webp", "images/ "} {
		var id string
		const insert = `INSERT INTO media_file (storage_key, mime_type, byte_size)
			VALUES ($1, 'image/webp', 10) RETURNING id::text`
		if err := conn.QueryRow(ctx, insert, key).Scan(&id); err != nil {
			t.Fatalf("valid new storage key %q: %v", key, err)
		}
		metadata := mediaMetadata(t, conn, ctx)
		if err := localStorageMigration(t, conn, ctx, "000005_media_local_storage.down.sql"); err == nil {
			t.Fatalf("downgrade accepted an unrepresentable legacy location %q", key)
		}
		assertMediaColumns(t, conn, ctx, true)
		var unchanged string
		if err := conn.QueryRow(ctx, `SELECT storage_key FROM media_file WHERE id = $1::uuid`, id).Scan(&unchanged); err != nil || unchanged != key {
			t.Fatalf("failed downgrade changed key: got=%q error=%v, want=%q", unchanged, err, key)
		}
		if got := mediaMetadata(t, conn, ctx); got != metadata {
			t.Fatal("failed downgrade changed media metadata")
		}
		fixed := fmt.Sprintf("fixed/image-%d.webp", i)
		localStorageExec(t, conn, ctx, `UPDATE media_file SET storage_key = $1 WHERE id = $2::uuid`, fixed, id)
		for _, file := range []string{"000005_media_local_storage.down.sql", "000005_media_local_storage.up.sql"} {
			if err := localStorageMigration(t, conn, ctx, file); err != nil {
				t.Fatalf("migration after repairing location: %v", err)
			}
		}
	}
}

func localStorageDatabase(t *testing.T) (*pgx.Conn, context.Context) {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set TEST_POSTGRES_DSN to test SQL migrations in an isolated schema")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect to test PostgreSQL: %v", err)
	}
	schema := pgx.Identifier{"media_test_" + strings.ReplaceAll(uuid.New().String(), "-", "")}.Sanitize()
	if _, err := conn.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		_ = conn.Close(ctx)
		t.Fatalf("create isolated test schema: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		defer conn.Close(cleanupCtx)
		_, _ = conn.Exec(cleanupCtx, "ROLLBACK")
		// The identifier is generated above, never supplied by a caller.
		if _, err := conn.Exec(cleanupCtx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Errorf("drop isolated test schema: %v", err)
		}
	})
	localStorageExec(t, conn, ctx, "SET search_path TO "+schema)
	for _, file := range []string{"000001_create_schema.up.sql", "000002_seed_account.up.sql", "000003_enforce_media_roles.up.sql", "000004_create_auth_session.up.sql"} {
		if err := localStorageMigration(t, conn, ctx, file); err != nil {
			t.Fatal(err)
		}
	}
	return conn, ctx
}

// The SQL files own BEGIN/COMMIT. On failure, roll back on the same connection
// before examining the old schema or retrying a migration.
func localStorageMigration(t *testing.T, conn *pgx.Conn, ctx context.Context, file string) error {
	t.Helper()
	sql, err := migrations.Files.ReadFile(file)
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}
	_, err = conn.Exec(ctx, string(sql), pgx.QueryExecModeSimpleProtocol)
	if err != nil {
		if _, rollbackErr := conn.Exec(ctx, "ROLLBACK"); rollbackErr != nil {
			t.Fatalf("roll back failed migration %s: %v (original error: %v)", file, rollbackErr, err)
		}
	}
	return err
}

func localStorageExec(t *testing.T, conn *pgx.Conn, ctx context.Context, query string, args ...any) {
	t.Helper()
	if _, err := conn.Exec(ctx, query, args...); err != nil {
		t.Fatalf("test SQL: %v", err)
	}
}

func legacyMedia(t *testing.T, conn *pgx.Conn, ctx context.Context, bucket, key string, image bool) string {
	t.Helper()
	mime, duration := "audio/ogg", any(int64(12345))
	if image {
		mime, duration = "image/webp", nil
	}
	const query = `INSERT INTO media_file (bucket, object_key, mime_type, byte_size, duration_ms, created_at, updated_at)
		VALUES ($1, $2, $3, 123, $4, '2026-01-01T00:00:00Z', '2026-01-02T00:00:00Z') RETURNING id::text`
	var id string
	if err := conn.QueryRow(ctx, query, bucket, key, mime, duration).Scan(&id); err != nil {
		t.Fatalf("create legacy media: %v", err)
	}
	return id
}

func mediaMetadata(t *testing.T, conn *pgx.Conn, ctx context.Context) string {
	t.Helper()
	var result string
	const query = `SELECT COALESCE(jsonb_agg(to_jsonb(m) ORDER BY id), '[]'::jsonb)::text
		FROM (SELECT id, mime_type, byte_size, duration_ms, created_at, updated_at FROM media_file) AS m`
	if err := conn.QueryRow(ctx, query).Scan(&result); err != nil {
		t.Fatal(err)
	}
	return result
}

func legacyMediaLocations(t *testing.T, conn *pgx.Conn, ctx context.Context) string {
	t.Helper()
	var result string
	const query = `SELECT COALESCE(jsonb_agg(to_jsonb(m) ORDER BY id), '[]'::jsonb)::text
		FROM (SELECT id, bucket, object_key FROM media_file) AS m`
	if err := conn.QueryRow(ctx, query).Scan(&result); err != nil {
		t.Fatal(err)
	}
	return result
}

func assertMediaColumns(t *testing.T, conn *pgx.Conn, ctx context.Context, upgraded bool) {
	t.Helper()
	var storage, bucket, object bool
	const query = `SELECT
		EXISTS (SELECT FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'media_file' AND column_name = 'storage_key'),
		EXISTS (SELECT FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'media_file' AND column_name = 'bucket'),
		EXISTS (SELECT FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'media_file' AND column_name = 'object_key')`
	if err := conn.QueryRow(ctx, query).Scan(&storage, &bucket, &object); err != nil {
		t.Fatal(err)
	}
	if storage != upgraded || bucket == upgraded || object == upgraded {
		t.Fatalf("wrong media schema: storage_key=%t bucket=%t object_key=%t, want upgraded=%t", storage, bucket, object, upgraded)
	}
}

func localStorageSQLState(t *testing.T, err error, code string) {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != code {
		t.Fatalf("SQL error=%v, want SQLSTATE %s", err, code)
	}
}
