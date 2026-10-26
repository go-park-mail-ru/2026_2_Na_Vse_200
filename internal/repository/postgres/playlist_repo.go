package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository"
)

const createPlaylistQuery = `
	INSERT INTO playlist (owner_id, title, description, cover_file_id, is_public)
	VALUES ($1::uuid, $2, $3, $4, $5)
	RETURNING id::text, revision, created_at, updated_at`

const getVisiblePlaylistQuery = `
	SELECT p.id::text, p.owner_id::text, p.title, p.description, p.cover_file_id::text,
		p.is_public, p.revision, p.created_at, p.updated_at,
		pi.track_id::text, pi.position
	FROM playlist AS p
	LEFT JOIN playlist_item AS pi ON pi.playlist_id = p.id
	WHERE p.id = $1::uuid AND (p.owner_id = $2::uuid OR p.is_public)
	ORDER BY pi.position`

const listOwnerPlaylistsQuery = `
	SELECT id::text, owner_id::text, title, description, cover_file_id::text,
		is_public, revision, created_at, updated_at
	FROM playlist
	WHERE owner_id = $1::uuid
	ORDER BY updated_at DESC, id
	LIMIT $2 OFFSET $3`

const replacePlaylistItemsQuery = `
	INSERT INTO playlist_item (playlist_id, track_id, position)
	SELECT $1::uuid, item.track_id::uuid, item.position
	FROM unnest($2::text[], $3::integer[]) AS item(track_id, position)`

const lockPlaylistForUpdateQuery = `
	SELECT revision FROM playlist
	WHERE id = $1::uuid AND owner_id = $2::uuid
	FOR UPDATE`

const updatePlaylistQuery = `
	UPDATE playlist
	SET title = $3, description = $4, cover_file_id = $5, is_public = $6, revision = revision + 1
	WHERE id = $1::uuid AND owner_id = $2::uuid
	RETURNING revision, updated_at`

const deletePlaylistItemsQuery = `DELETE FROM playlist_item WHERE playlist_id = $1::uuid`
const deletePlaylistQuery = `DELETE FROM playlist WHERE id = $1::uuid AND owner_id = $2::uuid`

// PlaylistRepo manages playlist metadata and ordered track membership.
type PlaylistRepo struct {
	pool *pgxpool.Pool
}

var _ repository.PlaylistRepositoryInterface = (*PlaylistRepo)(nil)

// NewPlaylistRepo creates a playlist repository over the shared pool.
func NewPlaylistRepo(pool *pgxpool.Pool) *PlaylistRepo {
	return &PlaylistRepo{pool: pool}
}

func (r *PlaylistRepo) Create(
	ctx context.Context,
	playlist models.Playlist,
) (result models.Playlist, resultErr error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.Playlist{}, fmt.Errorf("begin create playlist: %w", err)
	}
	defer rollback(ctx, tx, &resultErr)

	if err := tx.QueryRow(ctx, createPlaylistQuery,
		string(playlist.OwnerID), playlist.Title, playlist.Description,
		mediaFileIDValue(playlist.CoverFileID), playlist.IsPublic,
	).Scan(&playlist.ID, &playlist.Revision, &playlist.CreatedAt, &playlist.UpdatedAt); err != nil {
		return models.Playlist{}, fmt.Errorf("insert playlist: %w", err)
	}
	if err := insertPlaylistItems(ctx, tx, playlist.ID, playlist.Items); err != nil {
		return models.Playlist{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.Playlist{}, fmt.Errorf("commit create playlist: %w", err)
	}
	return playlist, nil
}

// GetVisible returns a playlist only to its owner or when it is public.
func (r *PlaylistRepo) GetVisible(ctx context.Context, id models.PlaylistID, viewerID models.UserID) (models.Playlist, error) {
	rows, err := r.pool.Query(ctx, getVisiblePlaylistQuery, string(id), string(viewerID))
	if err != nil {
		if isInvalidID(err) {
			return models.Playlist{}, repository.ErrPlaylistNotFound
		}
		return models.Playlist{}, fmt.Errorf("select visible playlist: %w", err)
	}
	defer rows.Close()

	var playlist models.Playlist
	found := false
	for rows.Next() {
		var trackID *string
		var coverID *string
		var position *int32
		if err := rows.Scan(
			&playlist.ID, &playlist.OwnerID, &playlist.Title, &playlist.Description,
			&coverID, &playlist.IsPublic, &playlist.Revision,
			&playlist.CreatedAt, &playlist.UpdatedAt, &trackID, &position,
		); err != nil {
			return models.Playlist{}, fmt.Errorf("scan visible playlist: %w", err)
		}
		found = true
		if coverID != nil {
			fileID := models.MediaFileID(*coverID)
			playlist.CoverFileID = &fileID
		}
		if trackID != nil && position != nil {
			playlist.Items = append(playlist.Items, models.PlaylistItem{
				TrackID: models.TrackID(*trackID), Position: int(*position),
			})
		}
	}
	if err := rows.Err(); err != nil {
		return models.Playlist{}, fmt.Errorf("read visible playlist: %w", err)
	}
	if !found {
		return models.Playlist{}, repository.ErrPlaylistNotFound
	}
	return playlist, nil
}

func (r *PlaylistRepo) ListByOwner(ctx context.Context, ownerID models.UserID, limit, offset int) ([]models.Playlist, error) {
	playlists := make([]models.Playlist, 0)
	if limit <= 0 {
		return playlists, nil
	}
	if offset < 0 {
		return nil, errors.New("playlist offset cannot be negative")
	}

	rows, err := r.pool.Query(ctx, listOwnerPlaylistsQuery, string(ownerID), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("select owner playlists: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var playlist models.Playlist
		var coverID *string
		if err := rows.Scan(
			&playlist.ID, &playlist.OwnerID, &playlist.Title, &playlist.Description,
			&coverID, &playlist.IsPublic, &playlist.Revision,
			&playlist.CreatedAt, &playlist.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan owner playlist: %w", err)
		}
		if coverID != nil {
			fileID := models.MediaFileID(*coverID)
			playlist.CoverFileID = &fileID
		}
		playlists = append(playlists, playlist)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read owner playlists: %w", err)
	}
	return playlists, nil
}

// Update replaces playlist metadata and its complete ordered contents in one transaction.
func (r *PlaylistRepo) Update(
	ctx context.Context,
	playlist models.Playlist,
	expectedRevision int64,
) (result models.Playlist, resultErr error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.Playlist{}, fmt.Errorf("begin update playlist: %w", err)
	}
	defer rollback(ctx, tx, &resultErr)

	var currentRevision int64
	err = tx.QueryRow(ctx, lockPlaylistForUpdateQuery, string(playlist.ID), string(playlist.OwnerID)).Scan(&currentRevision)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidID(err) {
			return models.Playlist{}, repository.ErrPlaylistNotFound
		}
		return models.Playlist{}, fmt.Errorf("lock playlist for update: %w", err)
	}
	if currentRevision != expectedRevision {
		return models.Playlist{}, repository.ErrRevisionConflict
	}

	err = tx.QueryRow(ctx, updatePlaylistQuery,
		string(playlist.ID), string(playlist.OwnerID), playlist.Title, playlist.Description,
		mediaFileIDValue(playlist.CoverFileID), playlist.IsPublic,
	).Scan(&playlist.Revision, &playlist.UpdatedAt)
	if err != nil {
		return models.Playlist{}, fmt.Errorf("update playlist metadata: %w", err)
	}
	if _, err := tx.Exec(ctx, deletePlaylistItemsQuery, string(playlist.ID)); err != nil {
		return models.Playlist{}, fmt.Errorf("delete playlist items: %w", err)
	}
	if err := insertPlaylistItems(ctx, tx, playlist.ID, playlist.Items); err != nil {
		return models.Playlist{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.Playlist{}, fmt.Errorf("commit update playlist: %w", err)
	}
	return playlist, nil
}

func (r *PlaylistRepo) Delete(ctx context.Context, id models.PlaylistID, ownerID models.UserID) error {
	tag, err := r.pool.Exec(ctx, deletePlaylistQuery, string(id), string(ownerID))
	if err != nil {
		return fmt.Errorf("delete playlist: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrPlaylistNotFound
	}
	return nil
}

func insertPlaylistItems(ctx context.Context, tx pgx.Tx, playlistID models.PlaylistID, items []models.PlaylistItem) error {
	if len(items) == 0 {
		return nil
	}
	trackIDs := make([]string, len(items))
	positions := make([]int32, len(items))
	for i, item := range items {
		trackIDs[i] = string(item.TrackID)
		if item.Position > 0 {
			if int64(item.Position) > math.MaxInt32 {
				return fmt.Errorf("playlist position %d is outside the positive integer range", item.Position)
			}
			positions[i] = int32(item.Position)
		} else {
			if int64(i+1) > math.MaxInt32 {
				return errors.New("playlist has more items than supported positions")
			}
			positions[i] = int32(i + 1)
		}
	}
	if _, err := tx.Exec(ctx, replacePlaylistItemsQuery, string(playlistID), trackIDs, positions); err != nil {
		return fmt.Errorf("insert playlist items: %w", err)
	}
	return nil
}
