package models

import "errors"

// Ошибки предметной области проверяются через errors.Is.
var (
	ErrMediaFileNotFound = errors.New("media file not found")
	ErrArtistNotFound    = errors.New("artist not found")
	ErrTrackNotFound     = errors.New("track not found")
	ErrUserNotFound      = errors.New("user not found")
	ErrEmailTaken        = errors.New("email already taken")
	ErrSessionNotFound   = errors.New("session not found")
	ErrSessionExpired    = errors.New("session expired")
)
