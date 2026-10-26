package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
	"uuid"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository/postgres"
)

func TestContentRepositories(t *testing.T) {
	pool, _ := sessionTestDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	owner := sessionTestUser(t, pool)
	viewer := sessionTestUser(t, pool)

	media := postgres.NewMediaRepo(pool)
	catalog := postgres.NewCatalogWriteRepo(pool)
	firstArtist, err := catalog.CreateArtist(ctx, models.Artist{Name: "Persistence artist", Biography: "Biography"})
	if err != nil {
		t.Fatalf("CreateArtist: %v", err)
	}
	secondArtist, err := catalog.CreateArtist(ctx, models.Artist{Name: "Second artist"})
	if err != nil {
		t.Fatalf("CreateArtist second: %v", err)
	}

	cover, err := media.Create(ctx, models.MediaFile{
		StorageKey: fmt.Sprintf("test/%s.webp", uuid.New().String()),
		MIMEType:   "image/webp",
		ByteSize:   128,
	})
	if err != nil {
		t.Fatalf("create cover media: %v", err)
	}
	cover, err = media.GetByID(ctx, cover.ID)
	if err != nil || cover.MIMEType != "image/webp" || cover.DurationMS != nil {
		t.Fatalf("GetByID cover = %+v, error = %v", cover, err)
	}

	duration := int64(185000)
	track, err := catalog.CreateTrackWithAudio(ctx, models.Track{
		UploaderID:  owner.ID,
		Title:       "First track",
		CoverFileID: mediaIDPointer(cover.ID),
		Artists:     []models.ArtistRef{{ID: firstArtist.ID}},
	}, models.MediaFile{
		StorageKey: fmt.Sprintf("test/%s.ogg", uuid.New().String()),
		MIMEType:   "audio/ogg",
		ByteSize:   2048,
		DurationMS: &duration,
	})
	if err != nil {
		t.Fatalf("CreateTrackWithAudio: %v", err)
	}
	invalidAudioKey := fmt.Sprintf("test/%s.webp", uuid.New().String())
	if _, err := catalog.CreateTrackWithAudio(ctx, models.Track{
		UploaderID: owner.ID,
		Title:      "Rejected track",
	}, models.MediaFile{
		StorageKey: invalidAudioKey,
		MIMEType:   "image/webp",
		ByteSize:   128,
	}); err == nil {
		t.Fatal("CreateTrackWithAudio accepted an image file as audio")
	}
	var invalidAudioPersisted bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM media_file WHERE storage_key = $1)`, invalidAudioKey).
		Scan(&invalidAudioPersisted); err != nil || invalidAudioPersisted {
		t.Fatalf("failed track insert left media metadata behind: persisted=%v error=%v", invalidAudioPersisted, err)
	}
	secondAudio, err := media.Create(ctx, models.MediaFile{
		StorageKey: fmt.Sprintf("test/%s.ogg", uuid.New().String()),
		MIMEType:   "audio/ogg",
		ByteSize:   1024,
		DurationMS: &duration,
	})
	if err != nil {
		t.Fatalf("create second audio: %v", err)
	}
	secondTrack, err := catalog.CreateTrack(ctx, models.Track{
		UploaderID:  owner.ID,
		Title:       "Second track",
		AudioFileID: secondAudio.ID,
		Status:      "published",
		Artists:     []models.ArtistRef{{ID: firstArtist.ID}},
	})
	if err != nil {
		t.Fatalf("CreateTrack: %v", err)
	}

	track.Title = "Edited track"
	track.Artists = []models.ArtistRef{{ID: secondArtist.ID}}
	track.Status = "published"
	if _, err := catalog.UpdateTrack(ctx, track); err != nil {
		t.Fatalf("UpdateTrack: %v", err)
	}

	releaseDate := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	album, err := catalog.CreateAlbum(ctx, models.Album{
		Title:       "Persistence album",
		ReleaseDate: &releaseDate,
		CoverFileID: mediaIDPointer(cover.ID),
		Artists:     []models.ArtistRef{{ID: secondArtist.ID}},
		Tracks: []models.AlbumTrack{
			{TrackID: track.ID, TrackNumber: 1},
			{TrackID: secondTrack.ID, TrackNumber: 2},
		},
	})
	if err != nil {
		t.Fatalf("CreateAlbum: %v", err)
	}
	album.Title = "Edited album"
	album.Tracks = []models.AlbumTrack{{TrackID: secondTrack.ID, TrackNumber: 1}}
	if _, err := catalog.UpdateAlbum(ctx, album); err != nil {
		t.Fatalf("UpdateAlbum: %v", err)
	}

	profileRepo := postgres.NewUserRepo(pool)
	updated, err := profileRepo.UpdateProfile(ctx, owner.ID, owner.Email, "Updated display name", mediaIDPointer(cover.ID))
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if updated.DisplayName != "Updated display name" || updated.AvatarFileID == nil || *updated.AvatarFileID != cover.ID {
		t.Fatalf("updated profile = %+v", updated)
	}

	playlists := postgres.NewPlaylistRepo(pool)
	playlist, err := playlists.Create(ctx, models.Playlist{
		OwnerID: owner.ID,
		Title:   "Private playlist",
		Items: []models.PlaylistItem{
			{TrackID: track.ID, Position: 1},
			{TrackID: track.ID, Position: 2},
			{TrackID: secondTrack.ID, Position: 3},
		},
	})
	if err != nil {
		t.Fatalf("Create playlist: %v", err)
	}
	if _, err := playlists.GetVisible(ctx, playlist.ID, viewer.ID); !errors.Is(err, repository.ErrPlaylistNotFound) {
		t.Fatalf("private playlist visible to another account: %v", err)
	}
	gotPlaylist, err := playlists.GetVisible(ctx, playlist.ID, owner.ID)
	if err != nil || len(gotPlaylist.Items) != 3 || gotPlaylist.Items[1].TrackID != track.ID {
		t.Fatalf("GetVisible playlist = %+v, error = %v", gotPlaylist, err)
	}
	playlist.Title = "Public playlist"
	playlist.IsPublic = true
	playlist.Items = []models.PlaylistItem{{TrackID: secondTrack.ID, Position: 1}}
	playlist, err = playlists.Update(ctx, playlist, playlist.Revision)
	if err != nil || playlist.Revision != 1 {
		t.Fatalf("Update playlist = %+v, error = %v", playlist, err)
	}
	invalidUpdate := playlist
	invalidUpdate.Items = []models.PlaylistItem{{TrackID: models.TrackID(uuid.New().String()), Position: 1}}
	if _, err := playlists.Update(ctx, invalidUpdate, playlist.Revision); err == nil {
		t.Fatal("Update playlist accepted a missing track")
	}
	persistedPlaylist, err := playlists.GetVisible(ctx, playlist.ID, owner.ID)
	if err != nil || persistedPlaylist.Revision != 1 || persistedPlaylist.Title != "Public playlist" ||
		len(persistedPlaylist.Items) != 1 || persistedPlaylist.Items[0].TrackID != secondTrack.ID {
		t.Fatalf("failed playlist update was not rolled back: %+v, error = %v", persistedPlaylist, err)
	}
	if _, err := playlists.Update(ctx, playlist, 0); !errors.Is(err, repository.ErrRevisionConflict) {
		t.Fatalf("stale playlist update error = %v, want revision conflict", err)
	}
	if gotPlaylist, err := playlists.GetVisible(ctx, playlist.ID, viewer.ID); err != nil || !gotPlaylist.IsPublic {
		t.Fatalf("public playlist read = %+v, error = %v", gotPlaylist, err)
	}
	ownerPlaylists, err := playlists.ListByOwner(ctx, owner.ID, 10, 0)
	if err != nil || len(ownerPlaylists) != 1 {
		t.Fatalf("ListByOwner = %v, error = %v", ownerPlaylists, err)
	}

	favorites := postgres.NewFavoriteRepo(pool)
	if err := favorites.Add(ctx, owner.ID, track.ID); err != nil {
		t.Fatalf("Add favorite: %v", err)
	}
	if err := favorites.Add(ctx, owner.ID, track.ID); err != nil {
		t.Fatalf("Add duplicate favorite: %v", err)
	}
	favoriteIDs, err := favorites.List(ctx, owner.ID, 10, 0)
	if err != nil || len(favoriteIDs) != 1 || favoriteIDs[0] != track.ID {
		t.Fatalf("favorite IDs = %v, error = %v", favoriteIDs, err)
	}
	if err := favorites.Remove(ctx, owner.ID, track.ID); err != nil {
		t.Fatalf("Remove favorite: %v", err)
	}

	history := postgres.NewListeningRepo(pool)
	for _, trackID := range []models.TrackID{track.ID, track.ID, secondTrack.ID} {
		if _, err := history.Record(ctx, models.ListeningEvent{AccountID: owner.ID, TrackID: trackID}); err != nil {
			t.Fatalf("Record listening event: %v", err)
		}
	}
	events, err := history.ListByAccount(ctx, owner.ID, 10, 0)
	if err != nil || len(events) != 3 {
		t.Fatalf("ListByAccount history = %v, error = %v", events, err)
	}

	now := time.Now().UTC()
	charts := postgres.NewChartRepo(pool)
	chart, err := charts.CreateFromHistory(ctx, models.Chart{
		Name:        "Test chart",
		PeriodStart: now.Add(-time.Hour),
		PeriodEnd:   now.Add(time.Hour),
	}, 10)
	if err != nil {
		t.Fatalf("CreateFromHistory: %v", err)
	}
	if len(chart.Items) != 2 || chart.Items[0].TrackID != track.ID || chart.Items[0].StreamsCount != 2 {
		t.Fatalf("generated chart items = %+v", chart.Items)
	}
	loadedChart, err := charts.GetByID(ctx, chart.ID)
	if err != nil || len(loadedChart.Items) != 2 || loadedChart.Items[1].Position != 2 {
		t.Fatalf("GetByID chart = %+v, error = %v", loadedChart, err)
	}
	chartList, err := charts.List(ctx, 10, 0)
	if err != nil || len(chartList) != 1 || chartList[0].ID != chart.ID {
		t.Fatalf("List charts = %v, error = %v", chartList, err)
	}

	if err := playlists.Delete(ctx, playlist.ID, owner.ID); err != nil {
		t.Fatalf("Delete playlist: %v", err)
	}
}

func mediaIDPointer(id models.MediaFileID) *models.MediaFileID {
	return &id
}
