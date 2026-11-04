package handlers

import (
	"context"
	"io"
	"os"
	"time"

	"2026_2_Na_Vse_200/internal/auth"
	"2026_2_Na_Vse_200/internal/config"
	"2026_2_Na_Vse_200/internal/delivery/handlers/media"
	"2026_2_Na_Vse_200/internal/models"
	artistusecase "2026_2_Na_Vse_200/internal/usecase/artist"
	authusecase "2026_2_Na_Vse_200/internal/usecase/auth"
	catalogusecase "2026_2_Na_Vse_200/internal/usecase/catalog"
	"2026_2_Na_Vse_200/internal/usecase/ports"
	profileusecase "2026_2_Na_Vse_200/internal/usecase/profile"
)

type repositoryTestDeps struct {
	Users    ports.UserRepository
	Sessions ports.SessionRepository
	Catalog  ports.CatalogRepository
	Artists  ports.ArtistRepository
	Media    ports.MediaRepository
	Blobs    profileBlobStore
	Hasher   ports.PasswordHasher
}

type profileBlobStore interface {
	Save(key string, r io.Reader) error
	Remove(key string) error
	Open(key string) (*os.File, error)
}

type memoryBlobs struct {
	files map[string][]byte
}

func (s *memoryBlobs) Save(key string, r io.Reader) error {
	if s.files == nil {
		s.files = map[string][]byte{}
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	s.files[key] = data
	return nil
}

func (s *memoryBlobs) Remove(key string) error {
	delete(s.files, key)
	return nil
}

func (s *memoryBlobs) Open(string) (*os.File, error) {
	return nil, os.ErrNotExist
}

type memoryMedia struct {
	files map[models.MediaFileID]models.MediaFile
}

func (s *memoryMedia) Create(_ context.Context, file models.MediaFile) (models.MediaFile, error) {
	if s.files == nil {
		s.files = map[models.MediaFileID]models.MediaFile{}
	}
	file.ID = models.MediaFileID("55555555-5555-5555-5555-555555555555")
	s.files[file.ID] = file
	return file, nil
}

func (s *memoryMedia) GetByID(_ context.Context, id models.MediaFileID) (models.MediaFile, error) {
	file, ok := s.files[id]
	if !ok {
		return models.MediaFile{}, models.ErrMediaFileNotFound
	}
	return file, nil
}

func newRepositoryTestAPI(cfg *config.Config, deps *repositoryTestDeps) *API {
	artists := deps.Artists
	if artists == nil {
		artists = artistRepoStub{}
	}
	mediaRepo := deps.Media
	if mediaRepo == nil {
		mediaRepo = &memoryMedia{}
	}
	blobs := deps.Blobs
	if blobs == nil {
		blobs = &memoryBlobs{}
	}

	var profiles ports.ProfileRepository
	if value, ok := deps.Users.(ports.ProfileRepository); ok {
		profiles = value
	}

	return New(cfg, &Deps{
		Auth:     authusecase.New(deps.Users, deps.Sessions, deps.Hasher, auth.NewSessionID, time.Now, cfg.SessionTTL),
		Catalog:  catalogusecase.New(deps.Catalog),
		Profile:  profileusecase.New(deps.Users, profiles, mediaRepo, blobs),
		Artist:   artistusecase.New(artists),
		Media:    media.NewMediaHandler(mediaRepo, blobs),
		Sessions: deps.Sessions,
	})
}
