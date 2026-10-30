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

type ArtistRepo struct{ pool *pgxpool.Pool }

var _ ports.ArtistRepository = (*ArtistRepo)(nil)

func NewArtistRepo(pool *pgxpool.Pool) *ArtistRepo { return &ArtistRepo{pool: pool} }

func (r *ArtistRepo) GetByID(ctx context.Context, id models.ArtistID) (models.Artist, error) {
	const query = `
 SELECT id::text, name, biography, image_file_id::text
 FROM artist WHERE id = $1::uuid`
	var artist models.Artist
	var image *string
	err := r.pool.QueryRow(ctx, query, string(id)).Scan(&artist.ID, &artist.Name, &artist.Biography, &image)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidID(err) {
		return models.Artist{}, models.ErrArtistNotFound
	}
	if err != nil {
		return models.Artist{}, fmt.Errorf("read artist: %w", err)
	}
	if image != nil {
		fileID := models.MediaFileID(*image)
		artist.ImageFileID = &fileID
	}
	return artist, nil
}

// Сначала ограничиваем список треков, затем присоединяем всех их исполнителей.
func (r *ArtistRepo) Tracks(ctx context.Context, id models.ArtistID, limit, offset int) ([]models.Track, error) {
	const query = `
 WITH selected_tracks AS (
  SELECT t.id, t.title, t.audio_file_id, t.created_at
  FROM track t
  JOIN track_artist ta ON ta.track_id = t.id
  WHERE ta.artist_id = $1::uuid AND t.status = 'published'
  ORDER BY t.created_at DESC, t.id LIMIT $2 OFFSET $3
 )
 SELECT t.id::text, t.title, audio.duration_ms, a.id::text, a.name
 FROM selected_tracks t
 JOIN media_file audio ON audio.id = t.audio_file_id
 LEFT JOIN track_artist ta ON ta.track_id = t.id
 LEFT JOIN artist a ON a.id = ta.artist_id
 ORDER BY t.created_at DESC, t.id, a.name, a.id`
	if limit <= 0 || offset < 0 {
		return nil, fmt.Errorf("invalid pagination")
	}
	return readTracks(ctx, r.pool, query, string(id), limit, offset)
}

func (r *ArtistRepo) Albums(ctx context.Context, id models.ArtistID, limit, offset int) ([]models.Album, error) {
	const query = `
 WITH selected_albums AS (
  SELECT al.id, al.title, al.release_date, al.created_at
  FROM album al
  JOIN album_artist aa ON aa.album_id = al.id
  WHERE aa.artist_id = $1::uuid
  ORDER BY al.created_at DESC, al.id LIMIT $2 OFFSET $3
 )
 SELECT al.id::text, al.title, al.release_date, a.id::text, a.name
 FROM selected_albums al
 LEFT JOIN album_artist aa ON aa.album_id = al.id
 LEFT JOIN artist a ON a.id = aa.artist_id
 ORDER BY al.created_at DESC, al.id, a.name, a.id`
	if limit <= 0 || offset < 0 {
		return nil, fmt.Errorf("invalid pagination")
	}
	return readAlbums(ctx, r.pool, query, string(id), limit, offset)
}
