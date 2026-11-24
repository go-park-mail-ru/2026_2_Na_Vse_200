package models

import "errors"

// Errors returned when reading or importing catalog entities.
var (
	ErrPlaylistNotFound      = errors.New("playlist not found")
	ErrPlaylistImportDeleted = errors.New("imported playlist was deleted")
	ErrAlbumNotFound         = errors.New("album not found")
)

// PlaylistPage contains the playlist metadata and its published tracks.
type PlaylistPage struct {
	ID          PlaylistID
	Title       string
	Description string
	CoverFileID *MediaFileID
	TrackCount  int
	Tracks      []Track
}

// PlaylistImportResult records a created or previously imported playlist.
type PlaylistImportResult struct {
	Playlist        PlaylistPage
	SkippedTrackIDs []string
	Created         bool
}

// PlaylistCoverPrefix identifies private playlist images in local storage.
const PlaylistCoverPrefix = "images/playlists/"

// PlaylistInput contains editable playlist fields and an optional cover.
type PlaylistInput struct {
	Name        string
	Description string
	Cover       []byte
	CoverMIME   string
}

// PlaylistTrackChange specifies tracks to append or a track to remove.
type PlaylistTrackChange struct {
	Add    []string
	Remove TrackID
}

// AlbumPage contains album metadata and its ordered published tracks.
type AlbumPage struct {
	Album  Album
	Tracks []Track
}

// SearchResult contains matching entities grouped by kind.
type SearchResult struct {
	Tracks  []Track
	Artists []Artist
	Albums  []Album
}

// SearchKind identifies the catalog entities included in a search.
type SearchKind string

// Supported catalog search kinds.
const (
	SearchAll     SearchKind = "all"
	SearchTracks  SearchKind = "tracks"
	SearchArtists SearchKind = "artists"
	SearchAlbums  SearchKind = "albums"
)

// SearchQuery contains the text, entity kind, and pagination of a catalog search.
type SearchQuery struct {
	Text   string
	Kind   SearchKind
	Limit  int
	Offset int
}
