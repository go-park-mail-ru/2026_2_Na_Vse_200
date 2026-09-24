// Package storage описывает, как backend обращается к хранилищу данных.
//
// Здесь только интерфейсы и словарь ошибок — договорённость между backend (Андрей)
// и БД (Фёдор). Реализацию на pgx пишет Фёдор в подпакете; обработчики зависят
// от интерфейсов, а не от PostgreSQL, поэтому их можно покрыть тестами с подставным
// хранилищем без поднятия базы.
//
// Правила для реализации:
//   - ошибки драйвера наружу не выпускаются: их место занимают значения ниже,
//     обёрнутые через fmt.Errorf("...: %w", err);
//   - вызывающий проверяет ошибки только через errors.Is, потому что после
//     оборачивания сравнение err == ErrX уже ложно;
//   - все методы принимают context.Context: по нему отменяется запрос, если клиент
//     ушёл или истёк таймаут.
//
// Контракт API: docs/api.md.
package storage

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
)

// Ошибки хранилища, которые обработчик обязан различать: по ним выбирается код ответа.
var (
	// ErrUserNotFound — пользователя с таким ID или email нет. Обработчик входа
	// отвечает на неё тем же 401, что и на неверный пароль.
	ErrUserNotFound = errors.New("user not found")

	// ErrEmailTaken — нарушено требование уникальности account.email. Возвращается
	// самой вставкой, а не отдельной проверкой «есть ли такой email»: иначе два
	// одновременных запроса создадут два аккаунта. Обработчик отвечает 409.
	ErrEmailTaken = errors.New("email already taken")

	// ErrSessionNotFound — сессии с таким идентификатором нет. Обработчик отвечает 401.
	ErrSessionNotFound = errors.New("session not found")

	// ErrSessionExpired — сессия найдена, но её срок истёк. Обработчик отвечает
	// тем же 401: снаружи это неотличимо от отсутствия сессии.
	ErrSessionExpired = errors.New("session expired")
)

// UserStorage — работа с аккаунтами, таблица account.
type UserStorage interface {
	// Create сохраняет пользователя и возвращает его с проставленными ID и CreatedAt.
	// Поле PasswordHash приходит уже заполненным: хеширование — дело backend, не БД.
	// Занятый email — ErrEmailTaken.
	Create(ctx context.Context, user models.User) (models.User, error)

	// GetByID возвращает пользователя по идентификатору. Нет такого — ErrUserNotFound.
	GetByID(ctx context.Context, id models.ID) (models.User, error)

	// GetByEmail возвращает пользователя по email вместе с PasswordHash: он нужен
	// для проверки пароля при входе. Email приходит уже в нижнем регистре.
	// Нет такого — ErrUserNotFound.
	GetByEmail(ctx context.Context, email string) (models.User, error)
}

// SessionStorage — работа с сессиями.
//
// Где именно они лежат, интерфейс не определяет. К РК1 реализация держит их
// в памяти процесса (мапа под sync.RWMutex): сессия не бизнес-сущность, в схеме БД
// её нет, отдельный сервис ради неё к 7 октября не поднимаем. В РК2 хранилище
// переезжает в Redis вместе с состоянием плеера — меняется только реализация,
// интерфейс и обработчики остаются прежними. См. docs/api.md, раздел 2.
type SessionStorage interface {
	// Create сохраняет новую сессию. Идентификатор и срок жизни задаёт backend.
	Create(ctx context.Context, session models.Session) error

	// GetByID возвращает сессию по идентификатору из cookie.
	// Нет такой — ErrSessionNotFound; просроченная — ErrSessionExpired.
	GetByID(ctx context.Context, id string) (models.Session, error)

	// Delete удаляет сессию при выходе. Удаление несуществующей сессии — не ошибка:
	// повторный выход обязан отвечать 204.
	Delete(ctx context.Context, id string) error
}

// CatalogStorage — чтение каталога для главной страницы.
//
// Методы отдают уже собранные карточки: длительность приходит из media_file,
// исполнители — из связующих таблиц track_artist и album_artist. Разбирать эти
// связи в обработчике не нужно.
type CatalogStorage interface {
	// HomeTracks возвращает не более limit опубликованных треков, новые первыми.
	// Пустой каталог — пустой срез и nil, а не ошибка.
	HomeTracks(ctx context.Context, limit int) ([]models.Track, error)

	// HomeArtists возвращает не более limit исполнителей, новые первыми.
	HomeArtists(ctx context.Context, limit int) ([]models.Artist, error)

	// HomeAlbums возвращает не более limit альбомов, новые первыми.
	HomeAlbums(ctx context.Context, limit int) ([]models.Album, error)
}
