package repository

import "errors"

// Ошибки, которые обработчик различает при выборе кода ответа.
// Проверяются через errors.Is: реализация вправе обернуть их контекстом.
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrEmailTaken        = errors.New("email already taken")
	ErrSessionNotFound   = errors.New("session not found")
	ErrSessionExpired    = errors.New("session expired")
	ErrMediaFileNotFound = errors.New("media file not found")
	ErrArtistNotFound    = errors.New("artist not found")
	ErrAlbumNotFound     = errors.New("album not found")
	ErrTrackNotFound     = errors.New("track not found")
	ErrPlaylistNotFound  = errors.New("playlist not found")
	ErrChartNotFound     = errors.New("chart not found")
	ErrRevisionConflict  = errors.New("revision conflict")
)
