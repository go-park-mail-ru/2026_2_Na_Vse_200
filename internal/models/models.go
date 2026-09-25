// Package models содержит доменные сущности проекта.
//
// Модели намеренно не знают ни про HTTP, ни про JSON: у них нет struct-тегов.
// Структуры ответов API описываются отдельно в слое обработчиков — так хеш пароля
// не может случайно уехать клиенту, а формат API можно менять, не трогая домен.
// Контракт API: docs/api.md.
package models

import "time"

// ID — идентификатор любой сущности каталога и аккаунта.
// Тип ключа в схеме БД ещё не зафиксирован (см. docs/api.md, раздел 9);
// в JSON идентификатор в любом случае отдаётся строкой.
type ID int64

// User — аккаунт пользователя, таблица account.
type User struct {
	ID           ID
	Email        string // хранится в нижнем регистре, уникален
	PasswordHash string // формат "алгоритм$соль$хеш", клиенту не отдаётся никогда
	DisplayName  string
	AvatarURL    string // пустая строка — аватара нет
	CreatedAt    time.Time
}

// Session — серверная сессия пользователя.
// Идентификатор непрозрачный: 32 случайных байта из crypto/rand в base64.
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

// Track — аудиозапись, таблица track.
// DurationMS приходит из media_file.duration_ms, в самой таблице track длительности нет.
type Track struct {
	ID         ID
	Title      string
	DurationMS int
	CoverURL   string      // пустая строка — обложки нет, клиент показывает заглушку
	Artists    []ArtistRef // у трека может быть несколько исполнителей, пустым не бывает
}

// Artist — исполнитель или группа, таблица artist.
type Artist struct {
	ID       ID
	Name     string
	ImageURL string // пустая строка — изображения нет
}

// Album — музыкальный релиз, таблица album.
type Album struct {
	ID          ID
	Title       string
	ReleaseDate *time.Time // nil — дата выпуска неизвестна
	CoverURL    string
	Artists     []ArtistRef
}
