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
