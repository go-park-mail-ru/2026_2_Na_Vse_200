package postgres

import (
	"context"
	"errors"
	"fmt"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TrackRepo struct{ pool *pgxpool.Pool }

var _ ports.TrackRepository = (*TrackRepo)(nil)

func NewTrackRepo(pool *pgxpool.Pool) *TrackRepo { return &TrackRepo{pool: pool} }

func (r *TrackRepo) GetByID(ctx context.Context, id models.TrackID) (models.Track, error) {
	const query = `
 SELECT t.id::text, t.uploader_id::text, t.title, t.audio_file_id::text,
        t.cover_file_id::text, t.status, audio.duration_ms
 FROM track t JOIN media_file audio ON audio.id = t.audio_file_id
 WHERE t.id = $1::uuid AND t.status = 'published'`
	var track models.Track
	var cover *string
	err := r.pool.QueryRow(ctx, query, string(id)).Scan(
		&track.ID, &track.UploaderID, &track.Title, &track.AudioFileID, &cover, &track.Status, &track.DurationMS)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidID(err) {
		return models.Track{}, models.ErrTrackNotFound
	}
	if err != nil {
		return models.Track{}, fmt.Errorf("read track: %w", err)
	}
	if cover != nil {
		fileID := models.MediaFileID(*cover)
		track.CoverFileID = &fileID
	}
	const artistsQuery = `
 SELECT a.id::text, a.name FROM artist a
 JOIN track_artist ta ON ta.artist_id = a.id
 WHERE ta.track_id = $1::uuid ORDER BY a.name, a.id`
	rows, err := r.pool.Query(ctx, artistsQuery, string(id))
	if err != nil {
		return models.Track{}, fmt.Errorf("read track artists: %w", err)
	}
	defer rows.Close()
	track.Artists = make([]models.ArtistRef, 0)
	for rows.Next() {
		var artist models.ArtistRef
		if err := rows.Scan(&artist.ID, &artist.Name); err != nil {
			return models.Track{}, fmt.Errorf("scan track artist: %w", err)
		}
		track.Artists = append(track.Artists, artist)
	}
	if err := rows.Err(); err != nil {
		return models.Track{}, fmt.Errorf("read track artists: %w", err)
	}
	return track, nil
}

// GetAudio выдаёт путь и метаданные аудио только опубликованного трека.
func (r *TrackRepo) GetAudio(ctx context.Context, id models.TrackID) (models.MediaFile, error) {
	const query = `
 SELECT m.id::text, m.storage_key, m.mime_type, m.byte_size, m.duration_ms, m.created_at, m.updated_at
 FROM track t JOIN media_file m ON m.id = t.audio_file_id
 WHERE t.id = $1::uuid AND t.status = 'published'`
	file, err := scanMediaFile(r.pool.QueryRow(ctx, query, string(id)))
	if errors.Is(err, models.ErrMediaFileNotFound) {
		return models.MediaFile{}, models.ErrTrackNotFound
	}
	return file, err
}

// FavoriteIDs возвращает ID опубликованных треков из избранного пользователя.
func (r *TrackRepo) FavoriteIDs(ctx context.Context, account models.UserID) ([]string, error) {
	const query = `
	 SELECT f.track_id::text
	 FROM favorite_track f
	 JOIN track t ON t.id = f.track_id
	 WHERE f.account_id = $1::uuid AND t.status = 'published'
	 ORDER BY f.track_id`

	rows, err := r.pool.Query(ctx, query, string(account))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}

		ids = append(ids, id)
	}

	return ids, rows.Err()
}

func (r *TrackRepo) SetFavorite(ctx context.Context, account models.UserID, track models.TrackID, liked bool) error {
	if !liked {
		const query = `
		 DELETE FROM favorite_track
		 WHERE account_id = $1::uuid AND track_id = $2::uuid`

		_, err := r.pool.Exec(ctx, query, string(account), string(track))
		return err
	}

	const query = `
	 WITH published AS (
	     SELECT id FROM track WHERE id = $2::uuid AND status = 'published'
	 ), inserted AS (
	     INSERT INTO favorite_track (account_id, track_id)
	     SELECT $1::uuid, id FROM published
	     ON CONFLICT DO NOTHING
	     RETURNING track_id
	 )
	 SELECT EXISTS(SELECT 1 FROM published)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, string(account), string(track)).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		return models.ErrTrackNotFound
	}

	return nil
}
