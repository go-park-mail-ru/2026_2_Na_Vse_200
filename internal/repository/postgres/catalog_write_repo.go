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

const createArtistQuery = `
	INSERT INTO artist (name, biography, image_file_id)
	VALUES ($1, $2, $3)
	RETURNING id::text`

const updateArtistQuery = `
	UPDATE artist
	SET name = $2, biography = $3, image_file_id = $4
	WHERE id = $1::uuid
	RETURNING id::text`

const createAlbumQuery = `
	INSERT INTO album (title, release_date, cover_file_id)
	VALUES ($1, $2, $3)
	RETURNING id::text`

const updateAlbumQuery = `
	UPDATE album
	SET title = $2, release_date = $3, cover_file_id = $4
	WHERE id = $1::uuid
	RETURNING id::text`

const createTrackQuery = `
	INSERT INTO track (uploader_id, title, audio_file_id, cover_file_id, status)
	VALUES ($1::uuid, $2, $3::uuid, $4, $5)
	RETURNING id::text`

const updateTrackQuery = `
	UPDATE track
	SET title = $2, audio_file_id = $3::uuid, cover_file_id = $4, status = $5
	WHERE id = $1::uuid
	RETURNING id::text`

const replaceTrackArtistsQuery = `
	INSERT INTO track_artist (track_id, artist_id)
	SELECT $1::uuid, artist_id::uuid
	FROM unnest($2::text[]) AS artists(artist_id)`

const replaceAlbumArtistsQuery = `
	INSERT INTO album_artist (album_id, artist_id)
	SELECT $1::uuid, artist_id::uuid
	FROM unnest($2::text[]) AS artists(artist_id)`

const replaceAlbumTracksQuery = `
	INSERT INTO album_track (album_id, track_id, track_number)
	SELECT $1::uuid, track_id::uuid, track_number
	FROM unnest($2::text[], $3::integer[]) AS item(track_id, track_number)`

const deleteAlbumArtistsQuery = `DELETE FROM album_artist WHERE album_id = $1::uuid`
const deleteAlbumTracksQuery = `DELETE FROM album_track WHERE album_id = $1::uuid`
const deleteTrackArtistsQuery = `DELETE FROM track_artist WHERE track_id = $1::uuid`

// CatalogWriteRepo atomically writes catalog entities and their relationships.
type CatalogWriteRepo struct {
	pool *pgxpool.Pool
}

var _ repository.CatalogWriteRepositoryInterface = (*CatalogWriteRepo)(nil)

// NewCatalogWriteRepo creates a catalog writer over the shared PostgreSQL pool.
func NewCatalogWriteRepo(pool *pgxpool.Pool) *CatalogWriteRepo {
	return &CatalogWriteRepo{pool: pool}
}

func (r *CatalogWriteRepo) CreateArtist(ctx context.Context, artist models.Artist) (models.Artist, error) {
	err := r.pool.QueryRow(ctx, createArtistQuery, artist.Name, artist.Biography, mediaFileIDValue(artist.ImageFileID)).Scan(&artist.ID)
	if err != nil {
		return models.Artist{}, fmt.Errorf("insert artist: %w", err)
	}
	return artist, nil
}

func (r *CatalogWriteRepo) UpdateArtist(ctx context.Context, artist models.Artist) (models.Artist, error) {
	err := r.pool.QueryRow(ctx, updateArtistQuery,
		string(artist.ID), artist.Name, artist.Biography, mediaFileIDValue(artist.ImageFileID),
	).Scan(&artist.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidID(err) {
			return models.Artist{}, repository.ErrArtistNotFound
		}
		return models.Artist{}, fmt.Errorf("update artist: %w", err)
	}
	return artist, nil
}

func (r *CatalogWriteRepo) CreateAlbum(ctx context.Context, album models.Album) (result models.Album, resultErr error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.Album{}, fmt.Errorf("begin create album: %w", err)
	}
	defer rollback(ctx, tx, &resultErr)

	if err := tx.QueryRow(ctx, createAlbumQuery,
		album.Title, album.ReleaseDate, mediaFileIDValue(album.CoverFileID),
	).Scan(&album.ID); err != nil {
		return models.Album{}, fmt.Errorf("insert album: %w", err)
	}
	if err := replaceAlbumRelations(ctx, tx, album); err != nil {
		return models.Album{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.Album{}, fmt.Errorf("commit create album: %w", err)
	}
	return album, nil
}

func (r *CatalogWriteRepo) UpdateAlbum(ctx context.Context, album models.Album) (result models.Album, resultErr error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.Album{}, fmt.Errorf("begin update album: %w", err)
	}
	defer rollback(ctx, tx, &resultErr)

	err = tx.QueryRow(ctx, updateAlbumQuery,
		string(album.ID), album.Title, album.ReleaseDate, mediaFileIDValue(album.CoverFileID),
	).Scan(&album.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidID(err) {
			return models.Album{}, repository.ErrAlbumNotFound
		}
		return models.Album{}, fmt.Errorf("update album: %w", err)
	}
	if _, err := tx.Exec(ctx, deleteAlbumArtistsQuery, string(album.ID)); err != nil {
		return models.Album{}, fmt.Errorf("delete album artists: %w", err)
	}
	if _, err := tx.Exec(ctx, deleteAlbumTracksQuery, string(album.ID)); err != nil {
		return models.Album{}, fmt.Errorf("delete album tracks: %w", err)
	}
	if err := replaceAlbumRelations(ctx, tx, album); err != nil {
		return models.Album{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.Album{}, fmt.Errorf("commit update album: %w", err)
	}
	return album, nil
}

func (r *CatalogWriteRepo) CreateTrack(ctx context.Context, track models.Track) (result models.Track, resultErr error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.Track{}, fmt.Errorf("begin create track: %w", err)
	}
	defer rollback(ctx, tx, &resultErr)

	if err := insertTrack(ctx, tx, &track); err != nil {
		return models.Track{}, err
	}
	if err := replaceTrackArtists(ctx, tx, track.ID, track.Artists); err != nil {
		return models.Track{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.Track{}, fmt.Errorf("commit create track: %w", err)
	}
	return track, nil
}

// CreateTrackWithAudio records the already-written audio file and track metadata
// in one database transaction. The filesystem write itself cannot be rolled back.
func (r *CatalogWriteRepo) CreateTrackWithAudio(
	ctx context.Context,
	track models.Track,
	audio models.MediaFile,
) (result models.Track, resultErr error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.Track{}, fmt.Errorf("begin create track with audio: %w", err)
	}
	defer rollback(ctx, tx, &resultErr)

	if err := tx.QueryRow(ctx, createMediaFileQuery,
		audio.StorageKey, audio.MIMEType, audio.ByteSize, audio.DurationMS,
	).Scan(&audio.ID, &audio.CreatedAt, &audio.UpdatedAt); err != nil {
		return models.Track{}, fmt.Errorf("insert track audio metadata: %w", err)
	}
	track.AudioFileID = audio.ID
	if err := insertTrack(ctx, tx, &track); err != nil {
		return models.Track{}, err
	}
	if err := replaceTrackArtists(ctx, tx, track.ID, track.Artists); err != nil {
		return models.Track{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.Track{}, fmt.Errorf("commit create track with audio: %w", err)
	}
	return track, nil
}

func (r *CatalogWriteRepo) UpdateTrack(ctx context.Context, track models.Track) (result models.Track, resultErr error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.Track{}, fmt.Errorf("begin update track: %w", err)
	}
	defer rollback(ctx, tx, &resultErr)

	err = tx.QueryRow(ctx, updateTrackQuery,
		string(track.ID), track.Title, string(track.AudioFileID), mediaFileIDValue(track.CoverFileID), trackStatus(track.Status),
	).Scan(&track.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidID(err) {
			return models.Track{}, repository.ErrTrackNotFound
		}
		return models.Track{}, fmt.Errorf("update track: %w", err)
	}
	if _, err := tx.Exec(ctx, deleteTrackArtistsQuery, string(track.ID)); err != nil {
		return models.Track{}, fmt.Errorf("delete track artists: %w", err)
	}
	if err := replaceTrackArtists(ctx, tx, track.ID, track.Artists); err != nil {
		return models.Track{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.Track{}, fmt.Errorf("commit update track: %w", err)
	}
	return track, nil
}

func insertTrack(ctx context.Context, tx pgx.Tx, track *models.Track) error {
	if err := tx.QueryRow(ctx, createTrackQuery,
		string(track.UploaderID), track.Title, string(track.AudioFileID),
		mediaFileIDValue(track.CoverFileID), trackStatus(track.Status),
	).Scan(&track.ID); err != nil {
		return fmt.Errorf("insert track: %w", err)
	}
	return nil
}

func replaceTrackArtists(ctx context.Context, tx pgx.Tx, trackID models.TrackID, artists []models.ArtistRef) error {
	if len(artists) == 0 {
		return nil
	}
	artistIDs := make([]string, len(artists))
	for i, artist := range artists {
		artistIDs[i] = string(artist.ID)
	}
	if _, err := tx.Exec(ctx, replaceTrackArtistsQuery, string(trackID), artistIDs); err != nil {
		return fmt.Errorf("insert track artists: %w", err)
	}
	return nil
}

func replaceAlbumRelations(ctx context.Context, tx pgx.Tx, album models.Album) error {
	if len(album.Artists) > 0 {
		artistIDs := make([]string, len(album.Artists))
		for i, artist := range album.Artists {
			artistIDs[i] = string(artist.ID)
		}
		if _, err := tx.Exec(ctx, replaceAlbumArtistsQuery, string(album.ID), artistIDs); err != nil {
			return fmt.Errorf("insert album artists: %w", err)
		}
	}
	if len(album.Tracks) > 0 {
		trackIDs := make([]string, len(album.Tracks))
		trackNumbers := make([]int32, len(album.Tracks))
		for i, track := range album.Tracks {
			if track.TrackNumber <= 0 || int64(track.TrackNumber) > math.MaxInt32 {
				return fmt.Errorf("album track number %d is outside the positive integer range", track.TrackNumber)
			}
			trackIDs[i] = string(track.TrackID)
			trackNumbers[i] = int32(track.TrackNumber)
		}
		if _, err := tx.Exec(ctx, replaceAlbumTracksQuery, string(album.ID), trackIDs, trackNumbers); err != nil {
			return fmt.Errorf("insert album tracks: %w", err)
		}
	}
	return nil
}

func mediaFileIDValue(id *models.MediaFileID) any {
	if id == nil {
		return nil
	}
	return string(*id)
}

func trackStatus(status string) string {
	if status == "" {
		return "draft"
	}
	return status
}

func rollback(ctx context.Context, tx pgx.Tx, resultErr *error) {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		*resultErr = errors.Join(*resultErr, fmt.Errorf("rollback transaction: %w", err))
	}
}
