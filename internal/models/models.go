// Package models содержит доменные сущности проекта.
package models

import "time"

// UserID — идентификатор сущности: UUID строкой, как первичные ключи в схеме БД.
type UserID string
type SessionID string
type ArtistID string
type TrackID string
type AlbumID string
type MediaFileID string
type PlaylistID string
type ChartID string
type ListeningEventID string

// User — аккаунт пользователя.
type User struct {
	ID           UserID
	Email        string // в нижнем регистре, уникален
	PasswordHash string // клиенту не отдаётся
	DisplayName  string
	AvatarURL    string // пустая строка — аватара нет
	AvatarFileID *MediaFileID
	CreatedAt    time.Time
}

// Session — серверная сессия пользователя.
type Session struct {
	ID        SessionID
	UserID    UserID
	ExpiresAt time.Time
}

// IsExpired сообщает, истекла ли сессия на момент now.
func (s Session) IsExpired(now time.Time) bool {
	return !now.Before(s.ExpiresAt)
}

// ArtistRef — краткие данные исполнителя внутри карточки трека или альбома.
type ArtistRef struct {
	ID   ArtistID
	Name string
}

// Track — аудиозапись.
type Track struct {
	ID          TrackID
	UploaderID  UserID
	Title       string
	AudioFileID MediaFileID
	CoverFileID *MediaFileID
	Status      string
	DurationMS  int
	CoverURL    string // пустая строка — обложки нет
	Artists     []ArtistRef
}

// Artist — исполнитель или группа.
type Artist struct {
	ID          ArtistID
	Name        string
	Biography   string
	ImageFileID *MediaFileID
	ImageURL    string // пустая строка — изображения нет
}

// Album — музыкальный релиз.
type Album struct {
	ID          AlbumID
	Title       string
	ReleaseDate *time.Time // nil — дата неизвестна
	CoverFileID *MediaFileID
	CoverURL    string
	Artists     []ArtistRef
	Tracks      []AlbumTrack
}

// MediaFile хранит метаданные файла, сами байты сохраняются файловым слоем.
type MediaFile struct {
	ID         MediaFileID
	StorageKey string
	MIMEType   string
	ByteSize   int64
	DurationMS *int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// AlbumTrack задаёт позицию трека в альбоме.
type AlbumTrack struct {
	TrackID     TrackID
	TrackNumber int
}

// Playlist — плейлист и его метаданные.
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

// PlaylistItem — трек в позиции плейлиста; повтор трека допустим.
type PlaylistItem struct {
	TrackID  TrackID
	Position int
}

// ListeningEvent фиксирует начало прослушивания.
type ListeningEvent struct {
	ID        ListeningEventID
	AccountID UserID
	TrackID   TrackID
	StartedAt time.Time
}

// Chart — сохранённый чарт за временной период.
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

// ChartItem хранит место и число прослушиваний трека в чарте.
type ChartItem struct {
	TrackID      TrackID
	Position     int
	StreamsCount int64
}
