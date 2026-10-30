package postgres_test

import (
	"context"
	"errors"
	"testing"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/repository/postgres"
)

// Использует отдельную схему через TEST_POSTGRES_DSN, как тесты сессий.
func TestProfileAvatar(t *testing.T) {
	pool, _ := sessionTestDatabase(t)
	ctx := context.Background()
	user := sessionTestUser(t, pool)
	users := postgres.NewUserRepo(pool)
	media := postgres.NewMediaRepo(pool)
	avatar, err := media.Create(ctx, models.MediaFile{StorageKey: "avatars/test.png", MIMEType: "image/png", ByteSize: 100})
	if err != nil {
		t.Fatal(err)
	}
	got, err := users.SetAvatar(ctx, user.ID, avatar.ID)
	if err != nil || got.AvatarFileID == nil || *got.AvatarFileID != avatar.ID {
		t.Fatalf("set avatar: %+v %v", got, err)
	}
	got, err = users.UpdateProfile(ctx, user.ID, "new@example.test", "New name")
	if err != nil || got.Email != "new@example.test" || got.DisplayName != "New name" || got.AvatarFileID == nil || *got.AvatarFileID != avatar.ID {
		t.Fatalf("update profile: %+v %v", got, err)
	}
	got, err = users.GetByID(ctx, user.ID)
	if err != nil || got.AvatarFileID == nil || *got.AvatarFileID != avatar.ID {
		t.Fatalf("read profile: %+v %v", got, err)
	}
	file, err := media.GetByID(ctx, avatar.ID)
	if err != nil || file.StorageKey != avatar.StorageKey || file.DurationMS != nil {
		t.Fatalf("read avatar: %+v %v", file, err)
	}
	other := sessionTestUser(t, pool)
	if _, err := users.UpdateProfile(ctx, user.ID, other.Email, "Name"); !errors.Is(err, models.ErrEmailTaken) {
		t.Fatalf("email conflict: %v", err)
	}
	for _, id := range []string{"not-uuid", "00000000-0000-0000-0000-000000000000"} {
		if _, err := users.UpdateProfile(ctx, models.UserID(id), "missing@example.test", "Name"); !errors.Is(err, models.ErrUserNotFound) {
			t.Fatalf("missing profile: %v", err)
		}
		if _, err := media.GetByID(ctx, models.MediaFileID(id)); !errors.Is(err, models.ErrMediaFileNotFound) {
			t.Fatalf("missing file: %v", err)
		}
	}
	if _, err := users.SetAvatar(ctx, user.ID, "00000000-0000-0000-0000-000000000000"); err == nil {
		t.Fatal("missing file accepted")
	}
}

func TestArtistAndPlayerRepositories(t *testing.T) {
	pool, _ := sessionTestDatabase(t)
	ctx := context.Background()
	user := sessionTestUser(t, pool)
	media := postgres.NewMediaRepo(pool)
	duration := int64(180000)
	audio, err := media.Create(ctx, models.MediaFile{StorageKey: "audio/test.mp3", MIMEType: "audio/mpeg", ByteSize: 1000, DurationMS: &duration})
	if err != nil {
		t.Fatal(err)
	}
	var artistID, otherID models.ArtistID
	if err := pool.QueryRow(ctx, "INSERT INTO artist (name, biography) VALUES ('Artist', 'Biography') RETURNING id::text").Scan(&artistID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "INSERT INTO artist (name) VALUES ('Other') RETURNING id::text").Scan(&otherID); err != nil {
		t.Fatal(err)
	}
	var published, draft models.TrackID
	for _, v := range []struct {
		status string
		id     *models.TrackID
	}{{"published", &published}, {"draft", &draft}} {
		err := pool.QueryRow(ctx, "INSERT INTO track (uploader_id, title, audio_file_id, status) VALUES ($1, 'Track', $2, $3) RETURNING id::text", string(user.ID), string(audio.ID), v.status).Scan(v.id)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, "INSERT INTO track_artist (track_id, artist_id) VALUES ($1, $2), ($1, $3)", string(*v.id), string(artistID), string(otherID)); err != nil {
			t.Fatal(err)
		}
	}
	var albumID models.AlbumID
	if err := pool.QueryRow(ctx, "INSERT INTO album (title) VALUES ('Album') RETURNING id::text").Scan(&albumID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "INSERT INTO album_artist (album_id,artist_id) VALUES ($1,$2),($1,$3)", string(albumID), string(artistID), string(otherID)); err != nil {
		t.Fatal(err)
	}
	artists := postgres.NewArtistRepo(pool)
	artist, err := artists.GetByID(ctx, artistID)
	if err != nil || artist.Biography != "Biography" || artist.ImageFileID != nil {
		t.Fatalf("artist: %+v %v", artist, err)
	}
	tracks, err := artists.Tracks(ctx, artistID, 1, 0)
	if err != nil || len(tracks) != 1 || tracks[0].ID != published || len(tracks[0].Artists) != 2 {
		t.Fatalf("tracks: %+v %v", tracks, err)
	}
	albums, err := artists.Albums(ctx, artistID, 1, 0)
	if err != nil || len(albums) != 1 || albums[0].ID != albumID || len(albums[0].Artists) != 2 {
		t.Fatalf("albums: %+v %v", albums, err)
	}
	tracks, err = artists.Tracks(ctx, artistID, 1, 1)
	if err != nil || len(tracks) != 0 {
		t.Fatalf("offset: %+v %v", tracks, err)
	}
	player := postgres.NewTrackRepo(pool)
	track, err := player.GetByID(ctx, published)
	if err != nil || track.AudioFileID != audio.ID || track.DurationMS != int(duration) || len(track.Artists) != 2 {
		t.Fatalf("track: %+v %v", track, err)
	}
	file, err := player.GetAudio(ctx, published)
	if err != nil || file.StorageKey != audio.StorageKey || file.MIMEType != "audio/mpeg" || file.ByteSize != 1000 {
		t.Fatalf("audio: %+v %v", file, err)
	}
	for _, id := range []models.TrackID{draft, "bad-id", "00000000-0000-0000-0000-000000000000"} {
		if _, err := player.GetByID(ctx, id); !errors.Is(err, models.ErrTrackNotFound) {
			t.Fatalf("hidden track: %v", err)
		}
		if _, err := player.GetAudio(ctx, id); !errors.Is(err, models.ErrTrackNotFound) {
			t.Fatalf("hidden audio: %v", err)
		}
	}
	if _, err := artists.GetByID(ctx, "bad-id"); !errors.Is(err, models.ErrArtistNotFound) {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := player.GetAudio(cancelled, published); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
}
