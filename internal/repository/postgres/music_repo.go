package postgres

import (
	"context"
	"strings"

	"2026_2_Na_Vse_200/internal/models"
)

const _trackProjection = `
	SELECT t.id::text, t.title, audio.duration_ms, t.cover_file_id::text, a.id::text, a.name
	FROM selected_tracks AS t
	JOIN media_file AS audio ON audio.id = t.audio_file_id
	LEFT JOIN track_artist AS ta ON ta.track_id = t.id
	LEFT JOIN artist AS a ON a.id = ta.artist_id`

const _albumProjection = `
	SELECT al.id::text, al.title, al.release_date, al.cover_file_id::text, a.id::text, a.name
	FROM selected_albums AS al
	LEFT JOIN album_artist AS aa ON aa.album_id = al.id
	LEFT JOIN artist AS a ON a.id = aa.artist_id`

// Explicit Cyrillic folding also works when the database uses the C locale.
const (
	_searchMatchPrefix = `NOT EXISTS (
		SELECT 1 FROM unnest($1::text[]) AS query(word)
		WHERE strpos(lower(translate(`
	_searchMatchSuffix = `,
			'АБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯё',
			'абвгдеежзийклмнопрстуфхцчшщъыьэюяе')), query.word) = 0
	)`
	_matchingTracks = `
		SELECT t.*
		FROM track t
		WHERE t.status = 'published' AND ` + _searchMatchPrefix + `concat_ws(' ', t.title, (
			SELECT string_agg(a.name, ' ')
			FROM track_artist ta JOIN artist a ON a.id = ta.artist_id
			WHERE ta.track_id = t.id
		))` + _searchMatchSuffix
)

// Search returns a page of matches for each requested catalog entity kind.
func (r *CatalogRepo) Search(ctx context.Context, query models.SearchQuery) (models.SearchResult, error) {
	ctx = WithDBOperation(ctx, "catalog.search")
	terms := strings.Fields(strings.ReplaceAll(strings.ToLower(query.Text), "ё", "е"))
	var result models.SearchResult
	if query.Kind == models.SearchAll || query.Kind == models.SearchTracks {
		tracks, err := readTracks(ctx, r.pool, `
			WITH selected_tracks AS (`+_matchingTracks+`
				ORDER BY t.created_at DESC, t.id LIMIT $2 OFFSET $3
			)`+_trackProjection+` ORDER BY t.created_at DESC, t.id, a.name, a.id`, terms, query.Limit, query.Offset,
		)
		if err != nil {
			return result, err
		}
		result.Tracks = tracks
	}
	if query.Kind == models.SearchAll || query.Kind == models.SearchArtists {
		artists, err := readArtists(ctx, r.pool, `
			WITH matching_tracks AS (`+_matchingTracks+`)
			SELECT a.id::text, a.name, a.image_file_id::text FROM artist a
			WHERE `+_searchMatchPrefix+`a.name`+_searchMatchSuffix+` OR EXISTS (
				SELECT 1 FROM track_artist ta JOIN matching_tracks t ON t.id = ta.track_id
				WHERE ta.artist_id = a.id
			)
			ORDER BY a.created_at DESC, a.id LIMIT $2 OFFSET $3`, terms, query.Limit, query.Offset,
		)
		if err != nil {
			return result, err
		}
		result.Artists = artists
	}
	if query.Kind == models.SearchAll || query.Kind == models.SearchAlbums {
		albums, err := readAlbums(ctx, r.pool, `
			WITH selected_albums AS (
				SELECT al.* FROM album al
				WHERE `+_searchMatchPrefix+`concat_ws(' ', al.title, (
					SELECT string_agg(a.name, ' ')
					FROM album_artist aa JOIN artist a ON a.id = aa.artist_id
					WHERE aa.album_id = al.id
				))`+_searchMatchSuffix+`
				ORDER BY al.created_at DESC, al.id LIMIT $2 OFFSET $3
			)`+_albumProjection+` ORDER BY al.created_at DESC, al.id, a.name, a.id`, terms, query.Limit, query.Offset,
		)
		if err != nil {
			return result, err
		}
		result.Albums = albums
	}
	return result, nil
}

// Album returns a catalog album and its published tracks in album order.
func (r *CatalogRepo) Album(ctx context.Context, id models.AlbumID) (models.AlbumPage, error) {
	ctx = WithDBOperation(ctx, "catalog.album")
	albums, err := readAlbums(ctx, r.pool, `
		WITH selected_albums AS (SELECT * FROM album WHERE id = $1::uuid)
		`+_albumProjection+` ORDER BY a.name, a.id`, string(id))
	if err != nil {
		return models.AlbumPage{}, err
	}
	if len(albums) == 0 {
		return models.AlbumPage{}, models.ErrAlbumNotFound
	}
	tracks, err := readTracks(ctx, r.pool, `
		WITH selected_tracks AS (
			SELECT t.*, at.track_number
			FROM album_track at JOIN track t ON t.id = at.track_id
			WHERE at.album_id = $1::uuid AND t.status = 'published'
		)`+_trackProjection+` ORDER BY t.track_number, a.name, a.id`, string(id))
	return models.AlbumPage{Album: albums[0], Tracks: tracks}, err
}

// FavoriteTracks returns the published tracks liked by the given account.
func (r *TrackRepo) FavoriteTracks(ctx context.Context, owner models.UserID) ([]models.Track, error) {
	ctx = WithDBOperation(ctx, "favorites.list")
	return readTracks(ctx, r.pool, `
		WITH selected_tracks AS (
			SELECT t.* FROM favorite_track f JOIN track t ON t.id = f.track_id
			WHERE f.account_id = $1::uuid AND t.status = 'published'
		)`+_trackProjection+` ORDER BY t.id, a.name, a.id`, string(owner))
}
