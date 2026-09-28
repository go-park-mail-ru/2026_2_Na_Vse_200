// Package models содержит доменные сущности проекта.
package models

import "time"

// UserID — идентификатор сущности: UUID строкой, как первичные ключи в схеме БД.
type UserID string
type SessionID string
type ArtistID string
type TrackID string
type AlbumID string

// User — аккаунт пользователя.
type User struct {
	ID           UserID
	Email        string // в нижнем регистре, уникален
	PasswordHash string // клиенту не отдаётся
	DisplayName  string
	AvatarURL    string // пустая строка — аватара нет
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
	ID         TrackID
	Title      string
	DurationMS int
	CoverURL   string // пустая строка — обложки нет
	Artists    []ArtistRef
}

// Artist — исполнитель или группа.
type Artist struct {
	ID       ArtistID
	Name     string
	ImageURL string // пустая строка — изображения нет
}

// Album — музыкальный релиз.
type Album struct {
	ID          AlbumID
	Title       string
	ReleaseDate *time.Time // nil — дата неизвестна
	CoverURL    string
	Artists     []ArtistRef
}
