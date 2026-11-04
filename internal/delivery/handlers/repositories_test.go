package handlers

import (
	"context"
	"time"
	"uuid"

	"2026_2_Na_Vse_200/internal/models"
)

type authUsersFixture struct {
	users []models.User
}

func (f *authUsersFixture) Create(_ context.Context, user models.User) (models.User, error) {
	for _, existing := range f.users {
		if existing.Email == user.Email {
			return models.User{}, models.ErrEmailTaken
		}
	}
	user.ID = models.UserID(uuid.New().String())
	user.CreatedAt = time.Now().UTC()
	f.users = append(f.users, user)
	return user, nil
}

func (f *authUsersFixture) GetByID(_ context.Context, id models.UserID) (models.User, error) {
	for _, user := range f.users {
		if user.ID == id {
			return user, nil
		}
	}
	return models.User{}, models.ErrUserNotFound
}

func (f *authUsersFixture) GetByEmail(_ context.Context, email string) (models.User, error) {
	for _, user := range f.users {
		if user.Email == email {
			return user, nil
		}
	}
	return models.User{}, models.ErrUserNotFound
}

type authSessionsFixture struct {
	sessions []models.Session
}

func (f *authSessionsFixture) Create(_ context.Context, session models.Session) error {
	f.sessions = append(f.sessions, session)
	return nil
}

func (f *authSessionsFixture) GetByID(_ context.Context, id models.SessionID) (models.Session, error) {
	for _, session := range f.sessions {
		if session.ID == id {
			if session.IsExpired(time.Now()) {
				return models.Session{}, models.ErrSessionExpired
			}
			return session, nil
		}
	}
	return models.Session{}, models.ErrSessionNotFound
}

func (f *authSessionsFixture) Delete(_ context.Context, id models.SessionID) error {
	for i, session := range f.sessions {
		if session.ID == id {
			f.sessions = append(f.sessions[:i], f.sessions[i+1:]...)
			break
		}
	}
	return nil
}

type homeCatalogFixture struct{}

func (homeCatalogFixture) HomeTracks(context.Context, int) ([]models.Track, error) {
	return []models.Track{{
		ID:         "3b8e5c90-77a1-4d2f-b6e4-12c9f0a5d738",
		Title:      "Тестовый трек",
		DurationMS: 123000,
		Artists:    []models.ArtistRef{{ID: "c41d7f02-9b63-4a58-8e17-0d5b2a6c9e34", Name: "Тестовый исполнитель"}},
	}}, nil
}

func (homeCatalogFixture) HomeArtists(context.Context, int) ([]models.Artist, error) {
	return []models.Artist{{ID: "c41d7f02-9b63-4a58-8e17-0d5b2a6c9e34", Name: "Тестовый исполнитель"}}, nil
}

func (homeCatalogFixture) HomeAlbums(context.Context, int) ([]models.Album, error) {
	return []models.Album{{ID: "5e0a94c6-2f18-4b7d-a390-6c81e7f2b405", Title: "Тестовый альбом"}}, nil
}

type artistRepoStub struct {
	artist models.Artist
	tracks []models.Track
	albums []models.Album
	err    error
}

func (s artistRepoStub) GetByID(context.Context, models.ArtistID) (models.Artist, error) {
	if s.err != nil {
		return models.Artist{}, s.err
	}
	if s.artist.ID == "" {
		return models.Artist{}, models.ErrArtistNotFound
	}
	return s.artist, nil
}

func (s artistRepoStub) Tracks(context.Context, models.ArtistID, int, int) ([]models.Track, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.tracks == nil {
		return []models.Track{}, nil
	}
	return s.tracks, nil
}

func (s artistRepoStub) Albums(context.Context, models.ArtistID, int, int) ([]models.Album, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.albums == nil {
		return []models.Album{}, nil
	}
	return s.albums, nil
}
