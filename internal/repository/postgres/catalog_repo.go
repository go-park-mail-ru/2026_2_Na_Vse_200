package postgres

import (
	"context"
	"fmt"
	"strings"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const _homeArtistsQuery = `
	SELECT id::text, name, image_file_id::text
	FROM artist
	ORDER BY created_at DESC, id
	LIMIT $1`

const _homeTracksQuery = `
	WITH selected_tracks AS (
		SELECT id, title, audio_file_id, cover_file_id, created_at
		FROM track
		WHERE status = 'published'
		ORDER BY created_at DESC, id
		LIMIT $1
	)
` + _trackProjection + `
	ORDER BY t.created_at DESC, t.id, a.name, a.id`

const _homeAlbumsQuery = `
	WITH selected_albums AS (
		SELECT id, title, release_date, cover_file_id, created_at
		FROM album
		ORDER BY created_at DESC, id
		LIMIT $1
	)
` + _albumProjection + `
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

// HomeArtists возвращает исполнителей, начиная с новых.
func (r *CatalogRepo) HomeArtists(ctx context.Context, limit int) ([]models.Artist, error) {
	artists := make([]models.Artist, 0)
	if limit <= 0 {
		return artists, nil
	}

	return readArtists(ctx, r.pool, _homeArtistsQuery, limit)
}

func readArtists(ctx context.Context, db catalogReader, query string, args ...any) ([]models.Artist, error) {
	artists := make([]models.Artist, 0)
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select artists: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			artist models.Artist
			id     string
			image  *string
		)
		if err := rows.Scan(&id, &artist.Name, &image); err != nil {
			return nil, fmt.Errorf("scan artist: %w", err)
		}

		artist.ID = models.ArtistID(id)
		if image != nil {
			fileID := models.MediaFileID(*image)
			artist.ImageFileID = &fileID
			artist.ImageURL = mediaURL(*image)
		}
		artists = append(artists, artist)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read artists: %w", err)
	}

	return artists, nil
}

// HomeTracks возвращает опубликованные треки с исполнителями, начиная с новых.
func (r *CatalogRepo) HomeTracks(ctx context.Context, limit int) ([]models.Track, error) {
	tracks := make([]models.Track, 0)
	if limit <= 0 {
		return tracks, nil
	}

	return readTracks(ctx, r.pool, _homeTracksQuery, limit)
}

type catalogReader interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func readTracks(ctx context.Context, pool catalogReader, query string, args ...any) ([]models.Track, error) {
	tracks := make([]models.Track, 0)
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select tracks: %w", err)
	}
	defer rows.Close()

	trackIndexes := make(map[models.TrackID]int)
	for rows.Next() {
		var (
			track                models.Track
			id                   string
			coverID              *string
			artistID, artistName *string
		)
		if err := rows.Scan(&id, &track.Title, &track.DurationMS, &coverID, &artistID, &artistName); err != nil {
			return nil, fmt.Errorf("scan track: %w", err)
		}
		track.ID = models.TrackID(id)
		if coverID != nil {
			fileID := models.MediaFileID(*coverID)
			track.CoverFileID = &fileID
			track.CoverURL = mediaURL(*coverID)
		}

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
		return nil, fmt.Errorf("read tracks: %w", err)
	}

	return tracks, nil
}

// HomeAlbums возвращает альбомы с исполнителями, начиная с новых.
func (r *CatalogRepo) HomeAlbums(ctx context.Context, limit int) ([]models.Album, error) {
	albums := make([]models.Album, 0)
	if limit <= 0 {
		return albums, nil
	}

	return readAlbums(ctx, r.pool, _homeAlbumsQuery, limit)
}

func readAlbums(ctx context.Context, pool catalogReader, query string, args ...any) ([]models.Album, error) {
	albums := make([]models.Album, 0)
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select albums: %w", err)
	}
	defer rows.Close()

	albumIndexes := make(map[models.AlbumID]int)
	for rows.Next() {
		var (
			album                models.Album
			id                   string
			coverID              *string
			artistID, artistName *string
		)
		if err := rows.Scan(&id, &album.Title, &album.ReleaseDate, &coverID, &artistID, &artistName); err != nil {
			return nil, fmt.Errorf("scan album: %w", err)
		}
		album.ID = models.AlbumID(id)
		if coverID != nil {
			fileID := models.MediaFileID(*coverID)
			album.CoverFileID = &fileID
			album.CoverURL = mediaURL(*coverID)
		}

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
		return nil, fmt.Errorf("read albums: %w", err)
	}

	return albums, nil
}

func mediaURL(fileID string) string {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" {
		return ""
	}
	return "/api/v1/media/" + fileID
}
