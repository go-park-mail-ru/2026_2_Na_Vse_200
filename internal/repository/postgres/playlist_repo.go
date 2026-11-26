package postgres

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"uuid"

	"2026_2_Na_Vse_200/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

const _playlistColumns = `p.id::text, p.title, p.description, p.cover_file_id::text`

type playlistBlobs interface {
	Save(string, io.Reader) error
	Remove(string) error
}

type playlistLock uint8

const (
	_playlistRead playlistLock = iota + 1
	_playlistForUpdate
)

type trackAvailability uint8

const (
	_requirePublishedTracks trackAvailability = iota + 1
	_skipUnavailableTracks
)

type playlistWrite struct {
	id       string
	legacyID string
	input    models.PlaylistInput
	trackIDs []string
}

// PlaylistRepo stores playlists in PostgreSQL and their cover images in local storage.
type PlaylistRepo struct {
	pool  *pgxpool.Pool
	blobs playlistBlobs
}

// NewPlaylistRepo creates a repository for playlists and cover images.
func NewPlaylistRepo(pool *pgxpool.Pool, blobs playlistBlobs) *PlaylistRepo {
	return &PlaylistRepo{pool: pool, blobs: blobs}
}

func scanPlaylist(row pgx.Row, target *models.PlaylistPage, extra ...any) error {
	fields := []any{&target.ID, &target.Title, &target.Description, &target.CoverFileID}
	err := row.Scan(append(fields, extra...)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.ErrPlaylistNotFound
	}
	return err
}

// List returns playlist cards for the given owner.
func (r *PlaylistRepo) List(ctx context.Context, owner models.UserID) ([]models.PlaylistPage, error) {
	ctx = WithDBOperation(ctx, "playlist.list")
	rows, err := r.pool.Query(ctx, `
		SELECT `+_playlistColumns+`, (
			SELECT count(*) FROM playlist_item pi JOIN track t ON t.id = pi.track_id
			WHERE pi.playlist_id = p.id AND t.status = 'published'
		)
		FROM playlist p WHERE p.owner_id = $1::uuid ORDER BY p.created_at DESC, p.id`, string(owner))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]models.PlaylistPage, 0)
	for rows.Next() {
		var playlist models.PlaylistPage
		if err := scanPlaylist(rows, &playlist, &playlist.TrackCount); err != nil {
			return nil, err
		}
		items = append(items, playlist)
	}
	return items, rows.Err()
}

func playlistForOwner(
	ctx context.Context,
	tx pgx.Tx,
	owner models.UserID,
	id string,
	lock playlistLock,
) (models.PlaylistPage, error) {
	var playlist models.PlaylistPage
	const query = `
		SELECT ` + _playlistColumns + ` FROM playlist p
		WHERE p.owner_id = $1::uuid AND p.id = COALESCE(
			(SELECT id FROM playlist WHERE owner_id = $1::uuid AND id = $2::uuid),
			(SELECT playlist_id FROM playlist_import WHERE owner_id = $1::uuid AND legacy_id = $3)
		)`
	var canonical *string
	if parsed, err := uuid.Parse(id); err == nil {
		value := parsed.String()
		canonical = &value
	}
	statement := query
	if lock == _playlistForUpdate {
		statement += " FOR UPDATE"
	}
	err := scanPlaylist(tx.QueryRow(ctx, statement, string(owner), canonical, id), &playlist)
	return playlist, err
}

func withPlaylistTracks(ctx context.Context, db catalogReader, playlist models.PlaylistPage) (models.PlaylistPage, error) {
	tracks, err := readTracks(ctx, db, `
		WITH selected_tracks AS (
			SELECT t.*, pi.position FROM playlist_item pi JOIN track t ON t.id = pi.track_id
			WHERE pi.playlist_id = $1::uuid AND t.status = 'published'
		)`+_trackProjection+` ORDER BY t.position, a.name, a.id`, string(playlist.ID))
	playlist.Tracks = tracks
	playlist.TrackCount = len(tracks)
	return playlist, err
}

// Get returns a playlist and its tracks by current or legacy ID.
func (r *PlaylistRepo) Get(ctx context.Context, owner models.UserID, id string) (models.PlaylistPage, error) {
	ctx = WithDBOperation(ctx, "playlist.get")
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return models.PlaylistPage{}, err
	}
	defer tx.Rollback(ctx)
	playlist, err := playlistForOwner(ctx, tx, owner, id, _playlistRead)
	if err != nil {
		return playlist, err
	}
	return withPlaylistTracks(ctx, tx, playlist)
}

// Save creates a playlist or updates an existing one when id is provided.
func (r *PlaylistRepo) Save(
	ctx context.Context,
	owner models.UserID,
	id string,
	input models.PlaylistInput,
) (models.PlaylistPage, error) {
	ctx = WithDBOperation(ctx, "playlist.save")
	result, err := r.save(ctx, owner, playlistWrite{id: id, input: input})
	return result.Playlist, err
}

// Import creates a playlist once for each owner and legacy ID.
func (r *PlaylistRepo) Import(
	ctx context.Context,
	owner models.UserID,
	legacyID string,
	input models.PlaylistInput,
	trackIDs []string,
) (models.PlaylistImportResult, error) {
	ctx = WithDBOperation(ctx, "playlist.import")
	return r.save(ctx, owner, playlistWrite{legacyID: legacyID, input: input, trackIDs: trackIDs})
}

func (r *PlaylistRepo) save(
	ctx context.Context,
	owner models.UserID,
	write playlistWrite,
) (models.PlaylistImportResult, error) {
	var result models.PlaylistImportResult
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return result, fmt.Errorf("begin playlist write: %w", err)
	}
	defer tx.Rollback(ctx)

	if write.legacyID != "" {
		tag, err := tx.Exec(ctx, `
			INSERT INTO playlist_import (owner_id, legacy_id) VALUES ($1::uuid, $2)
			ON CONFLICT DO NOTHING`, string(owner), write.legacyID)
		if err != nil {
			return result, fmt.Errorf("reserve playlist import: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return readPlaylistImport(ctx, tx, owner, write.legacyID)
		}
	}

	var page models.PlaylistPage
	if write.id != "" {
		page, err = playlistForOwner(ctx, tx, owner, write.id, _playlistForUpdate)
		if err != nil {
			return result, err
		}
	}

	previousCover := page.CoverFileID
	newCover, discardCover, err := r.saveCover(ctx, tx, write.input)
	defer func() {
		if discardCover != nil {
			discardCover()
		}
	}()
	if err != nil {
		return result, err
	}

	if newCover != "" {
		// saveCover returns the metadata ID in the playlist transaction.
		page.CoverFileID = &newCover
	}
	if err := writePlaylist(ctx, tx, owner, &page, write.input); err != nil {
		return result, err
	}

	if write.legacyID != "" {
		result.SkippedTrackIDs, err = appendPlaylistTracks(ctx, tx, page.ID, write.trackIDs, _skipUnavailableTracks)
		if err != nil {
			return result, err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE playlist_import SET playlist_id = $3::uuid, skipped_track_ids = $4
			WHERE owner_id = $1::uuid AND legacy_id = $2`,
			string(owner), write.legacyID, string(page.ID), result.SkippedTrackIDs,
		); err != nil {
			return result, fmt.Errorf("complete playlist import: %w", err)
		}
	}
	result.Playlist, err = withPlaylistTracks(ctx, tx, page)
	if err != nil {
		return result, err
	}

	var oldKey string
	if newCover != "" {
		oldKey, err = deletePlaylistCover(ctx, tx, previousCover)
		if err != nil {
			return result, err
		}
	}
	err = tx.Commit(ctx)
	if err == nil || !errors.Is(err, pgx.ErrTxCommitRollback) {
		// A connection failure can leave the commit outcome unknown. Keep the new
		// blob so any committed playlist still has its cover.
		discardCover = nil
	}
	if err != nil {
		return result, fmt.Errorf("commit playlist write: %w", err)
	}
	if oldKey != "" {
		r.removeBlob(ctx, oldKey)
	}
	result.Created = write.id == ""
	return result, nil
}

func readPlaylistImport(
	ctx context.Context,
	tx pgx.Tx,
	owner models.UserID,
	legacyID string,
) (models.PlaylistImportResult, error) {
	var (
		canonical *string
		result    models.PlaylistImportResult
	)
	if err := tx.QueryRow(ctx, `
		SELECT playlist_id::text, skipped_track_ids FROM playlist_import
		WHERE owner_id = $1::uuid AND legacy_id = $2`, string(owner), legacyID).
		Scan(&canonical, &result.SkippedTrackIDs); err != nil {
		return result, fmt.Errorf("read playlist import: %w", err)
	}
	if canonical == nil {
		return result, models.ErrPlaylistImportDeleted
	}
	page, err := playlistForOwner(ctx, tx, owner, *canonical, _playlistForUpdate)
	if errors.Is(err, models.ErrPlaylistNotFound) {
		return result, models.ErrPlaylistImportDeleted
	}
	if err != nil {
		return result, err
	}
	result.Playlist, err = withPlaylistTracks(ctx, tx, page)
	return result, err
}

func (r *PlaylistRepo) saveCover(
	ctx context.Context,
	tx pgx.Tx,
	input models.PlaylistInput,
) (models.MediaFileID, func(), error) {
	if len(input.Cover) == 0 {
		return "", nil, nil
	}
	extension := ".jpg"
	if input.CoverMIME == "image/png" {
		extension = ".png"
	}
	key := models.PlaylistCoverPrefix + uuid.New().String() + extension
	if err := r.blobs.Save(key, bytes.NewReader(input.Cover)); err != nil {
		return "", nil, fmt.Errorf("save playlist cover: %w", err)
	}
	discard := func() { r.removeBlob(ctx, key) }

	var fileID models.MediaFileID
	if err := tx.QueryRow(ctx, `
		INSERT INTO media_file (storage_key, mime_type, byte_size) VALUES ($1, $2, $3)
		RETURNING id::text`, key, input.CoverMIME, len(input.Cover),
	).Scan(&fileID); err != nil {
		discard()
		return "", nil, fmt.Errorf("insert playlist cover metadata: %w", err)
	}
	return fileID, discard, nil
}

func writePlaylist(
	ctx context.Context,
	tx pgx.Tx,
	owner models.UserID,
	page *models.PlaylistPage,
	input models.PlaylistInput,
) error {
	if page.ID == "" {
		return scanPlaylist(tx.QueryRow(ctx, `
			INSERT INTO playlist AS p (owner_id, title, description, cover_file_id)
			VALUES ($1::uuid, $2, $3, $4::uuid) RETURNING `+_playlistColumns,
			string(owner), input.Name, input.Description, page.CoverFileID,
		), page)
	}
	return scanPlaylist(tx.QueryRow(ctx, `
		UPDATE playlist AS p
		SET title = $2, description = $3, cover_file_id = $4::uuid, revision = revision + 1
		WHERE p.id = $1::uuid RETURNING `+_playlistColumns,
		string(page.ID), input.Name, input.Description, page.CoverFileID,
	), page)
}

func appendPlaylistTracks(
	ctx context.Context,
	tx pgx.Tx,
	playlist models.PlaylistID,
	ids []string,
	availability trackAvailability,
) ([]string, error) {
	available := make([]string, 0, len(ids))
	skipped := make([]string, 0)
	seen := make(map[string]bool, len(ids))
	for _, raw := range ids {
		id, err := uuid.Parse(raw)
		if err != nil {
			if availability == _requirePublishedTracks {
				return nil, models.ErrTrackNotFound
			}
			if !seen[raw] {
				skipped = append(skipped, raw)
				seen[raw] = true
			}
			continue
		}
		if key := id.String(); !seen[key] {
			seen[key] = true
			available = append(available, key)
		}
	}

	const query = `
		WITH requested AS (
			SELECT id, position FROM unnest($2::uuid[]) WITH ORDINALITY AS request(id, position)
		), inserted AS (
			INSERT INTO playlist_item (playlist_id, track_id, position)
			SELECT $1::uuid, t.id, baseline.last_position + row_number() OVER (ORDER BY r.position)
			FROM requested r JOIN track t ON t.id = r.id AND t.status = 'published'
			CROSS JOIN (
				SELECT COALESCE(max(position), 0) AS last_position FROM playlist_item WHERE playlist_id = $1::uuid
			) AS baseline
			WHERE NOT EXISTS (
				SELECT 1 FROM playlist_item pi WHERE pi.playlist_id = $1::uuid AND pi.track_id = t.id
			)
		)
		SELECT COALESCE(array_agg(r.id::text ORDER BY r.position), '{}')
		FROM requested r
		WHERE NOT EXISTS (SELECT 1 FROM track t WHERE t.id = r.id AND t.status = 'published')`
	var unavailable []string
	if err := tx.QueryRow(ctx, query, string(playlist), available).Scan(&unavailable); err != nil {
		return nil, err
	}
	if availability == _requirePublishedTracks && len(unavailable) > 0 {
		return nil, models.ErrTrackNotFound
	}
	return append(skipped, unavailable...), nil
}

// ChangeTracks appends or removes tracks while holding the playlist row lock.
func (r *PlaylistRepo) ChangeTracks(
	ctx context.Context,
	owner models.UserID,
	id string,
	change models.PlaylistTrackChange,
) (models.PlaylistPage, error) {
	ctx = WithDBOperation(ctx, "playlist.tracks.change")
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.PlaylistPage{}, err
	}
	defer tx.Rollback(ctx)
	playlist, err := playlistForOwner(ctx, tx, owner, id, _playlistForUpdate)
	if err != nil {
		return playlist, err
	}
	if change.Remove != "" {
		_, err = tx.Exec(ctx, `
			DELETE FROM playlist_item WHERE playlist_id = $1::uuid AND track_id = $2::uuid
		`, string(playlist.ID), string(change.Remove))
	} else {
		_, err = appendPlaylistTracks(ctx, tx, playlist.ID, change.Add, _requirePublishedTracks)
	}
	if err != nil {
		return playlist, err
	}
	if _, err := tx.Exec(ctx, "UPDATE playlist SET revision = revision + 1 WHERE id = $1::uuid", string(playlist.ID)); err != nil {
		return playlist, err
	}
	playlist, err = withPlaylistTracks(ctx, tx, playlist)
	if err != nil {
		return playlist, err
	}
	return playlist, tx.Commit(ctx)
}

// Delete removes a playlist and its uploaded cover.
func (r *PlaylistRepo) Delete(ctx context.Context, owner models.UserID, id string) error {
	ctx = WithDBOperation(ctx, "playlist.delete")
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	playlist, err := playlistForOwner(ctx, tx, owner, id, _playlistForUpdate)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "DELETE FROM playlist WHERE id = $1::uuid", string(playlist.ID)); err != nil {
		return err
	}
	key, err := deletePlaylistCover(ctx, tx, playlist.CoverFileID)
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit playlist deletion: %w", err)
	}
	if key != "" {
		r.removeBlob(ctx, key)
	}
	return nil
}

func deletePlaylistCover(ctx context.Context, tx pgx.Tx, cover *models.MediaFileID) (string, error) {
	if cover == nil {
		return "", nil
	}
	var key string
	err := tx.QueryRow(ctx, `
		DELETE FROM media_file WHERE id = $1::uuid AND storage_key LIKE $2
		RETURNING storage_key`, string(*cover), models.PlaylistCoverPrefix+"%",
	).Scan(&key)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("delete previous playlist cover: %w", err)
	}
	return key, nil
}

func (r *PlaylistRepo) removeBlob(ctx context.Context, key string) {
	if err := r.blobs.Remove(key); err != nil {
		zerolog.Ctx(ctx).Error().
			Str("operation", "playlist.cover.remove").
			Str("storage_key", key).
			Err(err).
			Msg("remove playlist cover")
	}
}

// Cover returns the cover metadata when the playlist belongs to the given owner.
func (r *PlaylistRepo) Cover(ctx context.Context, owner models.UserID, id, fileID string) (models.MediaFile, error) {
	ctx = WithDBOperation(ctx, "playlist.cover")
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return models.MediaFile{}, err
	}
	defer tx.Rollback(ctx)
	playlist, err := playlistForOwner(ctx, tx, owner, id, _playlistRead)
	if err != nil {
		return models.MediaFile{}, err
	}
	if playlist.CoverFileID == nil || string(*playlist.CoverFileID) != fileID {
		return models.MediaFile{}, models.ErrPlaylistNotFound
	}
	return scanMediaFile(tx.QueryRow(ctx, `
		SELECT id::text, storage_key, mime_type, byte_size, duration_ms, created_at, updated_at
		FROM media_file WHERE id = $1::uuid`, fileID))
}
