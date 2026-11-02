// Package repository объявляет интерфейсы доступа к данным.
// Ошибки хранилища лежат в errors.go.
package repository

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
)

// UserRepositoryInterface хранит аккаунты.
type UserRepositoryInterface interface {
	// Create сохраняет пользователя с уже посчитанным PasswordHash
	// и возвращает его с проставленными ID и CreatedAt.
	// Занятый email — ErrEmailTaken.
	Create(ctx context.Context, user models.User) (models.User, error)

	// GetByID возвращает пользователя или ErrUserNotFound.
	GetByID(ctx context.Context, id models.UserID) (models.User, error)

	// GetByEmail возвращает пользователя вместе с PasswordHash или ErrUserNotFound.
	// Email ожидается в нижнем регистре.
	GetByEmail(ctx context.Context, email string) (models.User, error)
}

// SessionRepositoryInterface хранит сессии.
type SessionRepositoryInterface interface {
	Create(ctx context.Context, session models.Session) error

	// GetByID возвращает сессию, ErrSessionNotFound или ErrSessionExpired.
	GetByID(ctx context.Context, id models.SessionID) (models.Session, error)

	// Delete удаляет сессию. Удаление несуществующей ошибкой не считается.
	Delete(ctx context.Context, id models.SessionID) error
}

// CatalogRepositoryInterface читает каталог для главной страницы.
// Методы возвращают готовые карточки: длительность и исполнители уже собраны.
type CatalogRepositoryInterface interface {
	// HomeTracks возвращает не более limit опубликованных треков, новые первыми.
	// Пустой каталог — пустой срез, а не ошибка.
	HomeTracks(ctx context.Context, limit int) ([]models.Track, error)

	// HomeArtists возвращает не более limit исполнителей, новые первыми.
	HomeArtists(ctx context.Context, limit int) ([]models.Artist, error)

	// HomeAlbums возвращает не более limit альбомов, новые первыми.
	HomeAlbums(ctx context.Context, limit int) ([]models.Album, error)
}

// ProfileRepositoryInterface редактирует публичные поля профиля аккаунта.
type ProfileRepositoryInterface interface {
	UpdateProfile(ctx context.Context, id models.UserID, email, displayName string, avatarFileID *models.MediaFileID) (models.User, error)
}

// MediaRepositoryInterface сохраняет и читает метаданные медиафайлов.
// Содержимое файла сохраняется отдельным файловым слоем.
type MediaRepositoryInterface interface {
	Create(ctx context.Context, file models.MediaFile) (models.MediaFile, error)
	GetByID(ctx context.Context, id models.MediaFileID) (models.MediaFile, error)
}

// CatalogWriteRepositoryInterface создаёт и изменяет записи музыкального каталога.
type CatalogWriteRepositoryInterface interface {
	CreateArtist(ctx context.Context, artist models.Artist) (models.Artist, error)
	UpdateArtist(ctx context.Context, artist models.Artist) (models.Artist, error)
	CreateAlbum(ctx context.Context, album models.Album) (models.Album, error)
	UpdateAlbum(ctx context.Context, album models.Album) (models.Album, error)
	CreateTrack(ctx context.Context, track models.Track) (models.Track, error)
	CreateTrackWithAudio(ctx context.Context, track models.Track, audio models.MediaFile) (models.Track, error)
	UpdateTrack(ctx context.Context, track models.Track) (models.Track, error)
}

// PlaylistRepositoryInterface управляет плейлистами и их составом.
type PlaylistRepositoryInterface interface {
	Create(ctx context.Context, playlist models.Playlist) (models.Playlist, error)
	GetVisible(ctx context.Context, id models.PlaylistID, viewerID models.UserID) (models.Playlist, error)
	ListByOwner(ctx context.Context, ownerID models.UserID, limit, offset int) ([]models.Playlist, error)
	Update(ctx context.Context, playlist models.Playlist, expectedRevision int64) (models.Playlist, error)
	Delete(ctx context.Context, id models.PlaylistID, ownerID models.UserID) error
}

// FavoriteRepositoryInterface управляет избранными треками аккаунта.
type FavoriteRepositoryInterface interface {
	Add(ctx context.Context, accountID models.UserID, trackID models.TrackID) error
	Remove(ctx context.Context, accountID models.UserID, trackID models.TrackID) error
	List(ctx context.Context, accountID models.UserID, limit, offset int) ([]models.TrackID, error)
}

// ListeningHistoryRepositoryInterface записывает и читает историю прослушиваний.
type ListeningHistoryRepositoryInterface interface {
	Record(ctx context.Context, event models.ListeningEvent) (models.ListeningEvent, error)
	ListByAccount(ctx context.Context, accountID models.UserID, limit, offset int) ([]models.ListeningEvent, error)
}

// ChartRepositoryInterface формирует сохранённые чарты из истории прослушиваний.
type ChartRepositoryInterface interface {
	CreateFromHistory(ctx context.Context, chart models.Chart, limit int) (models.Chart, error)
	GetByID(ctx context.Context, id models.ChartID) (models.Chart, error)
	List(ctx context.Context, limit, offset int) ([]models.Chart, error)
}
