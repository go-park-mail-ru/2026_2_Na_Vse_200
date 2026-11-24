package postgres_test

import (
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/repository/postgres"
	"2026_2_Na_Vse_200/internal/storage"
)

func TestRK3CatalogAndPlaylists(t *testing.T) {
	pool, _ := sessionTestDatabase(t)
	ctx := t.Context()
	owner := sessionTestUser(t, pool).ID
	other := sessionTestUser(t, pool).ID
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}

	const (
		audio   = "11000000-0000-4000-8000-000000000001"
		first   = "22000000-0000-4000-8000-000000000001"
		second  = "22000000-0000-4000-8000-000000000002"
		draft   = "22000000-0000-4000-8000-000000000003"
		artist  = "33000000-0000-4000-8000-000000000001"
		albumID = "44000000-0000-4000-8000-000000000001"
	)
	exec(`INSERT INTO media_file (id, storage_key, mime_type, byte_size, duration_ms)
	      VALUES ($1, 'audio/rk3.wav', 'audio/wav', 16, 2000)`, audio)
	exec(`INSERT INTO track (id, uploader_id, title, audio_file_id, status, created_at)
	      VALUES ($1, $4, 'Ёлочный мотив', $5, 'published', '2020-01-01'),
	             ($2, $4, 'Зимний рассвет', $5, 'published', '2020-01-01'),
	             ($3, $4, 'Ёлочный черновик', $5, 'draft', '2020-01-01')`, first, second, draft, string(owner), audio)
	exec(`INSERT INTO track (uploader_id, title, audio_file_id, status)
	      SELECT $1, 'Новый трек ' || n, $2, 'published' FROM generate_series(1, 23) n`, string(owner), audio)
	exec(`INSERT INTO artist (id, name) VALUES ($1, 'Семён Лес')`, artist)
	exec(`INSERT INTO track_artist (track_id, artist_id) VALUES ($1, $4), ($2, $4), ($3, $4)`, first, second, draft, artist)
	exec(`INSERT INTO album (id, title) VALUES ($1, 'Ёлки зимой')`, albumID)
	exec(`INSERT INTO album_artist (album_id, artist_id) VALUES ($1, $2)`, albumID, artist)
	exec(`INSERT INTO album_track (album_id, track_id, track_number) VALUES ($1, $2, 1), ($1, $3, 2), ($1, $4, 3)`, albumID, second, draft, first)

	catalog := postgres.NewCatalogRepo(pool)
	home, err := catalog.HomeTracks(ctx, 20)
	if err != nil || slices.Contains(trackIDs(home), first) {
		t.Fatalf("home: %v, %v", trackIDs(home), err)
	}
	matches, err := catalog.Search(ctx, models.SearchQuery{
		Text: "ЕЛОЧНЫЙ семен", Kind: models.SearchAll, Limit: 20,
	})
	if err != nil || !slices.Equal(trackIDs(matches.Tracks), []string{first}) || len(matches.Artists) != 1 {
		t.Fatalf("track and artist search: %+v, %v", matches, err)
	}
	matches, err = catalog.Search(ctx, models.SearchQuery{
		Text: "елки семён", Kind: models.SearchAlbums, Limit: 20,
	})
	if err != nil || len(matches.Albums) != 1 || len(matches.Albums[0].Artists) != 1 {
		t.Fatalf("album search: %+v, %v", matches, err)
	}
	matches, err = catalog.Search(ctx, models.SearchQuery{
		Text: "новый", Kind: models.SearchTracks, Limit: 20, Offset: 20,
	})
	if err != nil || len(matches.Tracks) != 3 {
		t.Fatalf("search offset: %d, %v", len(matches.Tracks), err)
	}
	album, err := catalog.Album(ctx, albumID)
	if err != nil || !slices.Equal(trackIDs(album.Tracks), []string{second, first}) || len(album.Album.Artists) != 1 {
		t.Fatalf("album: %+v, %v", album, err)
	}

	trackRepo := postgres.NewTrackRepo(pool)
	if err := trackRepo.SetFavorite(ctx, owner, first, true /* liked */); err != nil {
		t.Fatal(err)
	}
	favorites, err := trackRepo.FavoriteTracks(ctx, owner)
	if err != nil || !slices.Equal(trackIDs(favorites), []string{first}) || len(favorites[0].Artists) != 1 {
		t.Fatalf("favorites: %+v, %v", favorites, err)
	}

	blobs, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	playlists := postgres.NewPlaylistRepo(pool, blobs)
	input := models.PlaylistInput{Name: "Imported", Description: "description", Cover: []byte("first blob"), CoverMIME: "image/png"}
	imported, err := playlists.Import(ctx, owner, "m21-first", input, []string{second, draft, first, second})
	playlist := imported.Playlist
	if err != nil || !imported.Created || !slices.Equal(imported.SkippedTrackIDs, []string{draft}) ||
		!slices.Equal(trackIDs(playlist.Tracks), []string{second, first}) || playlist.CoverFileID == nil {
		t.Fatalf("import: %+v, %v", imported, err)
	}
	again, err := playlists.Import(ctx, owner, "m21-first", input, []string{draft})
	if err != nil || again.Created || again.Playlist.ID != playlist.ID ||
		!slices.Equal(again.SkippedTrackIDs, []string{draft}) {
		t.Fatalf("replayed import: %+v, %v", again, err)
	}
	got, err := playlists.Get(ctx, owner, "m21-first")
	if err != nil || got.ID != playlist.ID {
		t.Fatalf("legacy link: %+v %v", got, err)
	}
	if _, err := playlists.ChangeTracks(ctx, other, string(playlist.ID), models.PlaylistTrackChange{Add: []string{first}}); !errors.Is(err, models.ErrPlaylistNotFound) {
		t.Fatalf("foreign mutation: %v", err)
	}
	if _, err := playlists.Get(ctx, other, "m21-first"); !errors.Is(err, models.ErrPlaylistNotFound) {
		t.Fatalf("foreign legacy link: %v", err)
	}
	oldCover, err := playlists.Cover(ctx, owner, string(playlist.ID), string(*playlist.CoverFileID))
	if err != nil || !strings.HasPrefix(oldCover.StorageKey, models.PlaylistCoverPrefix) {
		t.Fatalf("original cover: %+v %v", oldCover, err)
	}
	input.Cover = []byte("updated blob")
	updated, err := playlists.Save(ctx, owner, "m21-first", input)
	if err != nil || updated.CoverFileID == nil || *updated.CoverFileID == *playlist.CoverFileID {
		t.Fatalf("replace cover: %+v %v", updated, err)
	}
	newCover, err := playlists.Cover(ctx, owner, string(playlist.ID), string(*updated.CoverFileID))
	if err != nil || newCover.ByteSize != int64(len(input.Cover)) {
		t.Fatalf("updated cover: %+v %v", newCover, err)
	}
	if _, err := playlists.Cover(ctx, other, string(playlist.ID), string(*updated.CoverFileID)); !errors.Is(err, models.ErrPlaylistNotFound) {
		t.Fatalf("foreign cover: %v", err)
	}
	if file, err := blobs.Open(oldCover.StorageKey); err == nil {
		file.Close()
		t.Fatal("replaced file still in storage")
	}

	empty, err := playlists.Save(ctx, owner, "", models.PlaylistInput{Name: "New"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := playlists.ChangeTracks(ctx, owner, string(empty.ID), models.PlaylistTrackChange{Add: []string{first, draft}}); !errors.Is(err, models.ErrTrackNotFound) {
		t.Fatalf("add unavailable: %v", err)
	}
	var (
		writers sync.WaitGroup
		results [2]error
	)
	for i := range results {
		writers.Go(func() {
			changed, err := playlists.ChangeTracks(ctx, owner, string(empty.ID), models.PlaylistTrackChange{
				Add: []string{first, first, second},
			})
			if err == nil && !slices.Equal(trackIDs(changed.Tracks), []string{first, second}) {
				err = errors.New("unexpected order or duplicate tracks")
			}
			results[i] = err
		})
	}
	writers.Wait()
	for _, err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}

	changed, err := playlists.ChangeTracks(ctx, owner, string(empty.ID), models.PlaylistTrackChange{Remove: first})
	if err != nil || !slices.Equal(trackIDs(changed.Tracks), []string{second}) {
		t.Fatalf("remove track: %+v %v", changed, err)
	}
	cards, err := playlists.List(ctx, owner)
	if err != nil || len(cards) != 2 || cards[0].ID != empty.ID || cards[0].TrackCount != 1 {
		t.Fatalf("playlist cards: %+v %v", cards, err)
	}
	if err := playlists.Delete(ctx, owner, string(playlist.ID)); err != nil {
		t.Fatal(err)
	}
	if _, err := playlists.Import(ctx, owner, "m21-first", input, nil); !errors.Is(err, models.ErrPlaylistImportDeleted) {
		t.Fatalf("replay after deletion: %v", err)
	}
	if data, err := blobs.Open(newCover.StorageKey); err == nil {
		data.Close()
		t.Fatal("deleted playlist left its cover in storage")
	}
}

func trackIDs(tracks []models.Track) []string {
	ids := make([]string, 0, len(tracks))
	for _, track := range tracks {
		ids = append(ids, string(track.ID))
	}
	return ids
}
