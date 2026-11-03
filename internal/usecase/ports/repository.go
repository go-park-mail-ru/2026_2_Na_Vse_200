package ports

import (
	"context"

	"2026_2_Na_Vse_200/internal/models"
)

// UserRepository хранит аккаунты.
type UserRepository interface {
	// Create сохраняет пользователя с готовым хешем пароля и возвращает ID и время создания.
	// Занятый email обозначается models.ErrEmailTaken.
	Create(ctx context.Context, user models.User) (models.User, error)

	// GetByID возвращает пользователя или models.ErrUserNotFound.
	GetByID(ctx context.Context, id models.UserID) (models.User, error)

	// GetByEmail ищет пользователя по нормализованному email и возвращает его с хешем пароля.
	GetByEmail(ctx context.Context, email string) (models.User, error)
}

// SessionRepository хранит сессии.
type SessionRepository interface {
	Create(ctx context.Context, session models.Session) error

	GetByID(ctx context.Context, id models.SessionID) (models.Session, error)

	// Delete удаляет сессию. Отсутствующий или некорректный ID не считается ошибкой.
	Delete(ctx context.Context, id models.SessionID) error
}

// CatalogRepository читает карточки каталога для главной страницы.
type CatalogRepository interface {
	// HomeTracks возвращает опубликованные треки с исполнителями, начиная с новых.
	HomeTracks(ctx context.Context, limit int) ([]models.Track, error)

	// HomeArtists возвращает исполнителей, начиная с новых.
	HomeArtists(ctx context.Context, limit int) ([]models.Artist, error)

	// HomeAlbums возвращает альбомы с исполнителями, начиная с новых.
	HomeAlbums(ctx context.Context, limit int) ([]models.Album, error)
}

type PasswordHasher interface {
	Hash(string) (string, error)
	Verify(string, string) (bool, error)
}
