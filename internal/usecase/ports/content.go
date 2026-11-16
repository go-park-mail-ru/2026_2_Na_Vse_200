package ports

import (
	"2026_2_Na_Vse_200/internal/models"
	"context"
)

// ProfileRepository изменяет данные профиля и привязку аватара.
type ProfileRepository interface {
	// UpdateProfile меняет имя; email должен совпадать с текущим и не изменяется.
	UpdateProfile(ctx context.Context, id models.UserID, email, displayName string) (models.User, error)
	SetAvatar(ctx context.Context, id models.UserID, fileID models.MediaFileID) (models.User, error)
}

// MediaRepository сохраняет метаданные файлов без их содержимого.
type MediaRepository interface {
	Create(ctx context.Context, file models.MediaFile) (models.MediaFile, error)
	GetByID(ctx context.Context, id models.MediaFileID) (models.MediaFile, error)
}

type ArtistRepository interface {
	GetByID(ctx context.Context, id models.ArtistID) (models.Artist, error)
	Tracks(ctx context.Context, id models.ArtistID, limit, offset int) ([]models.Track, error)
	Albums(ctx context.Context, id models.ArtistID, limit, offset int) ([]models.Album, error)
}

// TrackRepository читает опубликованные треки и метаданные их аудиофайлов.
type TrackRepository interface {
	GetByID(ctx context.Context, id models.TrackID) (models.Track, error)
	GetAudio(ctx context.Context, id models.TrackID) (models.MediaFile, error)
}
