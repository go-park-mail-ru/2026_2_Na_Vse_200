package models

import "time"

type UserID string
type SessionID string
type ArtistID string
type TrackID string
type AlbumID string
type MediaFileID string
type PlaylistID string
type ChartID string
type ListeningEventID string

type User struct {
	ID           UserID
	Email        string // Email хранится в нижнем регистре.
	PasswordHash string // Хеш не включается в ответ API.
	DisplayName  string
	AvatarURL    string // Пустая строка означает отсутствие изображения.
	AvatarFileID *MediaFileID
	CreatedAt    time.Time
}

type Session struct {
	ID        SessionID
	UserID    UserID
	ExpiresAt time.Time
}

func (s Session) IsExpired(now time.Time) bool {
	return !now.Before(s.ExpiresAt)
}

type ArtistRef struct {
	ID   ArtistID
	Name string
}

type Track struct {
	ID          TrackID
	UploaderID  UserID
	Title       string
	AudioFileID MediaFileID
	CoverFileID *MediaFileID
	Status      string
	DurationMS  int
	CoverURL    string
	Artists     []ArtistRef
}

type Artist struct {
	ID          ArtistID
	Name        string
	Biography   string
	ImageFileID *MediaFileID
	ImageURL    string
}

type Album struct {
	ID          AlbumID
	Title       string
	ReleaseDate *time.Time
	CoverFileID *MediaFileID
	CoverURL    string
	Artists     []ArtistRef
	Tracks      []AlbumTrack
}

type MediaFile struct {
	ID         MediaFileID
	StorageKey string
	MIMEType   string
	ByteSize   int64
	DurationMS *int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type AlbumTrack struct {
	TrackID     TrackID
	TrackNumber int
}

type Playlist struct {
	ID          PlaylistID
	OwnerID     UserID
	Title       string
	Description string
	CoverFileID *MediaFileID
	IsPublic    bool
	Revision    int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Items       []PlaylistItem
}

type PlaylistItem struct {
	TrackID  TrackID
	Position int
}

type ListeningEvent struct {
	ID        ListeningEventID
	AccountID UserID
	TrackID   TrackID
	StartedAt time.Time
}

type Chart struct {
	ID          ChartID
	Name        string
	PeriodStart time.Time
	PeriodEnd   time.Time
	CoverFileID *MediaFileID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Items       []ChartItem
}

type ChartItem struct {
	TrackID      TrackID
	Position     int
	StreamsCount int64
}
