package ports

import (
	"context"
	"2026_2_Na_Vse_200/internal/models"
)

// ProfileRepository меняет данные профиля. Читать профиль можно через UserRepository.GetByID.
type ProfileRepository interface {
	UpdateProfile(ctx context.Context, id models.UserID, email, displayName string) (models.User, error)
	SetAvatar(ctx context.Context, id models.UserID, fileID models.MediaFileID) (models.User, error)
}

// MediaRepository хранит сведения о файле. Сами байты сохраняет файловый слой.
type MediaRepository interface {
	Create(ctx context.Context, file models.MediaFile) (models.MediaFile, error)
	GetByID(ctx context.Context, id models.MediaFileID) (models.MediaFile, error)
}

type ArtistRepository interface {
	GetByID(ctx context.Context, id models.ArtistID) (models.Artist, error)
	Tracks(ctx context.Context, id models.ArtistID, limit, offset int) ([]models.Track, error)
	Albums(ctx context.Context, id models.ArtistID, limit, offset int) ([]models.Album, error)
}

// TrackRepository возвращает опубликованный трек и сведения об аудио для его выдачи.
type TrackRepository interface {
	GetByID(ctx context.Context, id models.TrackID) (models.Track, error)
	GetAudio(ctx context.Context, id models.TrackID) (models.MediaFile, error)
}
