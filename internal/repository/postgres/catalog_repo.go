package postgres

import (
	"context"
	"fmt"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
	"github.com/jackc/pgx/v5/pgxpool"
)

const homeArtistsQuery = `
	SELECT id::text, name
	FROM artist
	ORDER BY created_at DESC, id
	LIMIT $1`

const homeTracksQuery = `
	WITH selected_tracks AS (
		SELECT id, title, audio_file_id, created_at
		FROM track
		WHERE status = 'published'
		ORDER BY created_at DESC, id
		LIMIT $1
	)
	SELECT t.id::text, t.title, audio.duration_ms, a.id::text, a.name
	FROM selected_tracks AS t
	JOIN media_file AS audio ON audio.id = t.audio_file_id
	LEFT JOIN track_artist AS ta ON ta.track_id = t.id
	LEFT JOIN artist AS a ON a.id = ta.artist_id
	ORDER BY t.created_at DESC, t.id, a.name, a.id`

const homeAlbumsQuery = `
	WITH selected_albums AS (
		SELECT id, title, release_date, created_at
		FROM album
		ORDER BY created_at DESC, id
		LIMIT $1
	)
	SELECT al.id::text, al.title, al.release_date, a.id::text, a.name
	FROM selected_albums AS al
	LEFT JOIN album_artist AS aa ON aa.album_id = al.id
	LEFT JOIN artist AS a ON a.id = aa.artist_id
	ORDER BY al.created_at DESC, al.id, a.name, a.id`

// CatalogRepo читает музыкальный каталог из PostgreSQL.
type CatalogRepo struct {
	pool *pgxpool.Pool
}

var _ ports.CatalogRepository = (*CatalogRepo)(nil)

// NewCatalogRepo создаёт репозиторий поверх уже открытого пула.
func NewCatalogRepo(pool *pgxpool.Pool) *CatalogRepo {
	return &CatalogRepo{pool: pool}
}

// HomeArtists возвращает не более limit исполнителей, новые первыми.
func (r *CatalogRepo) HomeArtists(ctx context.Context, limit int) ([]models.Artist, error) {
	artists := make([]models.Artist, 0)
	if limit <= 0 {
		return artists, nil
	}

	rows, err := r.pool.Query(ctx, homeArtistsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("select home artists: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			artist models.Artist
			id     string
		)
		if err := rows.Scan(&id, &artist.Name); err != nil {
			return nil, fmt.Errorf("scan home artist: %w", err)
		}

		artist.ID = models.ArtistID(id)
		artists = append(artists, artist)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read home artists: %w", err)
	}

	return artists, nil
}

// HomeTracks возвращает не более limit опубликованных треков с исполнителями.
func (r *CatalogRepo) HomeTracks(ctx context.Context, limit int) ([]models.Track, error) {
	tracks := make([]models.Track, 0)
	if limit <= 0 {
		return tracks, nil
	}

	return readTracks(ctx, r.pool, homeTracksQuery, limit)
}

func readTracks(ctx context.Context, pool *pgxpool.Pool, query string, args ...any) ([]models.Track, error) {
	tracks := make([]models.Track, 0)
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select home tracks: %w", err)
	}
	defer rows.Close()

	trackIndexes := make(map[models.TrackID]int)
	for rows.Next() {
		var (
			track                models.Track
			id                   string
			artistID, artistName *string
		)
		if err := rows.Scan(&id, &track.Title, &track.DurationMS, &artistID, &artistName); err != nil {
			return nil, fmt.Errorf("scan home track: %w", err)
		}
		track.ID = models.TrackID(id)

		index, exists := trackIndexes[track.ID]
		if !exists {
			track.Artists = make([]models.ArtistRef, 0)
			index = len(tracks)
			trackIndexes[track.ID] = index
			tracks = append(tracks, track)
		}

		if artistID != nil && artistName != nil {
			tracks[index].Artists = append(tracks[index].Artists, models.ArtistRef{
				ID:   models.ArtistID(*artistID),
				Name: *artistName,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read home tracks: %w", err)
	}

	return tracks, nil
}

// HomeAlbums возвращает не более limit альбомов с датой релиза и исполнителями.
func (r *CatalogRepo) HomeAlbums(ctx context.Context, limit int) ([]models.Album, error) {
	albums := make([]models.Album, 0)
	if limit <= 0 {
		return albums, nil
	}

	return readAlbums(ctx, r.pool, homeAlbumsQuery, limit)
}

func readAlbums(ctx context.Context, pool *pgxpool.Pool, query string, args ...any) ([]models.Album, error) {
	albums := make([]models.Album, 0)
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select home albums: %w", err)
	}
	defer rows.Close()

	albumIndexes := make(map[models.AlbumID]int)
	for rows.Next() {
		var (
			album                models.Album
			id                   string
			artistID, artistName *string
		)
		if err := rows.Scan(&id, &album.Title, &album.ReleaseDate, &artistID, &artistName); err != nil {
			return nil, fmt.Errorf("scan home album: %w", err)
		}
		album.ID = models.AlbumID(id)

		index, exists := albumIndexes[album.ID]
		if !exists {
			album.Artists = make([]models.ArtistRef, 0)
			index = len(albums)
			albumIndexes[album.ID] = index
			albums = append(albums, album)
		}

		if artistID != nil && artistName != nil {
			albums[index].Artists = append(albums[index].Artists, models.ArtistRef{
				ID:   models.ArtistID(*artistID),
				Name: *artistName,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read home albums: %w", err)
	}

	return albums, nil
}
