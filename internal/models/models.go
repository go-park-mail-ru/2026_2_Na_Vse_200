// Package models содержит доменные сущности проекта.
// Struct-тегов здесь нет: форматы запросов и ответов описаны в слое обработчиков.
package models

import "time"

// ID — идентификатор сущности. В JSON отдаётся строкой.
type ID int64

// User — аккаунт пользователя.
type User struct {
	ID           ID
	Email        string // в нижнем регистре, уникален
	PasswordHash string // клиенту не отдаётся
	DisplayName  string
	AvatarURL    string // пустая строка — аватара нет
	CreatedAt    time.Time
}

// Session — серверная сессия пользователя.
type Session struct {
	ID        string
	UserID    ID
	ExpiresAt time.Time
}

// IsExpired сообщает, истекла ли сессия на момент now.
func (s Session) IsExpired(now time.Time) bool {
	return !now.Before(s.ExpiresAt)
}

// ArtistRef — краткие данные исполнителя внутри карточки трека или альбома.
type ArtistRef struct {
	ID   ID
	Name string
}

// Track — аудиозапись.
type Track struct {
	ID         ID
	Title      string
	DurationMS int
	CoverURL   string // пустая строка — обложки нет
	Artists    []ArtistRef
}

// Artist — исполнитель или группа.
type Artist struct {
	ID       ID
	Name     string
	ImageURL string // пустая строка — изображения нет
}

// Album — музыкальный релиз.
type Album struct {
	ID          ID
	Title       string
	ReleaseDate *time.Time // nil — дата неизвестна
	CoverURL    string
	Artists     []ArtistRef
}
