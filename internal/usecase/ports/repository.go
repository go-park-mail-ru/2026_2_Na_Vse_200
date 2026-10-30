// Package ports объявляет интерфейсы доступа к данным.
// Общие ошибки контрактов объявлены в internal/models/errors.go.
package ports

import (
	"context"

	"2026_2_Na_Vse_200/internal/models"
)

// UserRepository хранит аккаунты.
type UserRepository interface {
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

// SessionRepository хранит сессии.
type SessionRepository interface {
	Create(ctx context.Context, session models.Session) error

	// GetByID возвращает сессию, ErrSessionNotFound или ErrSessionExpired.
	GetByID(ctx context.Context, id models.SessionID) (models.Session, error)

	// Delete удаляет сессию. Удаление несуществующей ошибкой не считается.
	Delete(ctx context.Context, id models.SessionID) error
}

// CatalogRepository читает каталог для главной страницы.
// Методы возвращают готовые карточки: длительность и исполнители уже собраны.
type CatalogRepository interface {
	// HomeTracks возвращает не более limit опубликованных треков, новые первыми.
	// Пустой каталог — пустой срез, а не ошибка.
	HomeTracks(ctx context.Context, limit int) ([]models.Track, error)

	// HomeArtists возвращает не более limit исполнителей, новые первыми.
	HomeArtists(ctx context.Context, limit int) ([]models.Artist, error)

	// HomeAlbums возвращает не более limit альбомов, новые первыми.
	HomeAlbums(ctx context.Context, limit int) ([]models.Album, error)
}

type PasswordHasher interface {
	Hash(string) (string, error)
	Verify(string, string) (bool, error)
}
