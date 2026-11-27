package postgres_test

import (
	"context"
	"errors"
	"testing"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/repository/postgres"
)

func TestSaveAvatarTransaction(t *testing.T) {
	pool, _ := sessionTestDatabase(t)
	ctx := context.Background()
	repo := postgres.NewUserRepo(pool)
	user := sessionTestUser(t, pool)
	file := models.MediaFile{StorageKey: "avatars/atomic.png", MIMEType: "image/png", ByteSize: 10}
	updated, err := repo.SaveAvatar(ctx, user.ID, file)
	if err != nil || updated.AvatarFileID == nil {
		t.Fatalf("save avatar: %+v %v", updated, err)
	}
	stored, err := postgres.NewMediaRepo(pool).GetByID(ctx, *updated.AvatarFileID)
	if err != nil || stored.StorageKey != file.StorageKey {
		t.Fatalf("committed media: %+v %v", stored, err)
	}

	file.StorageKey = "avatars/rolled-back.png"
	_, err = repo.SaveAvatar(ctx, "ffffffff-ffff-4fff-8fff-ffffffffffff", file)
	if !errors.Is(err, models.ErrUserNotFound) {
		t.Fatalf("missing user: %v", err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM media_file WHERE storage_key=$1`, file.StorageKey).Scan(&count); err != nil || count != 0 {
		t.Fatalf("orphan media after rollback: count=%d err=%v", count, err)
	}
	current, err := repo.GetByID(ctx, user.ID)
	if err != nil || current.AvatarFileID == nil || *current.AvatarFileID != *updated.AvatarFileID {
		t.Fatalf("existing avatar changed: %+v %v", current, err)
	}
}
