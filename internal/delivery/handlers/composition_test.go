package handlers

import (
	"time"

	"2026_2_Na_Vse_200/internal/auth"
	"2026_2_Na_Vse_200/internal/config"
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
	Hasher   ports.PasswordHasher
}

func newRepositoryTestAPI(cfg *config.Config, deps *repositoryTestDeps) *API {
	artists := deps.Artists
	if artists == nil {
		artists = artistRepoStub{}
	}
	return New(cfg, &Deps{
		Auth:     authusecase.New(deps.Users, deps.Sessions, deps.Hasher, auth.NewSessionID, time.Now, cfg.SessionTTL),
		Catalog:  catalogusecase.New(deps.Catalog),
		Profile:  profileusecase.New(deps.Users),
		Artist:   artistusecase.New(artists),
		Sessions: deps.Sessions,
	})
}
