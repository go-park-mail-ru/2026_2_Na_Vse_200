// Package repository объявляет интерфейсы доступа к данным и ошибки хранилища.
package repository

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
)

// Ошибки, которые обработчик различает при выборе кода ответа.
// Проверяются через errors.Is: реализация вправе обернуть их контекстом.
var (
	ErrUserNotFound    = errors.New("user not found")
	ErrEmailTaken      = errors.New("email already taken")
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
)

// UserRepositoryInterface хранит аккаунты.
type UserRepositoryInterface interface {
	// Create сохраняет пользователя с уже посчитанным PasswordHash
	// и возвращает его с проставленными ID и CreatedAt.
	// Занятый email — ErrEmailTaken.
	Create(ctx context.Context, user models.User) (models.User, error)

	// GetByID возвращает пользователя или ErrUserNotFound.
	GetByID(ctx context.Context, id models.ID) (models.User, error)

	// GetByEmail возвращает пользователя вместе с PasswordHash или ErrUserNotFound.
	// Email ожидается в нижнем регистре.
	GetByEmail(ctx context.Context, email string) (models.User, error)
}

// SessionRepositoryInterface хранит сессии.
type SessionRepositoryInterface interface {
	Create(ctx context.Context, session models.Session) error

	// GetByID возвращает сессию, ErrSessionNotFound или ErrSessionExpired.
	GetByID(ctx context.Context, id string) (models.Session, error)

	// Delete удаляет сессию. Удаление несуществующей ошибкой не считается.
	Delete(ctx context.Context, id string) error
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
