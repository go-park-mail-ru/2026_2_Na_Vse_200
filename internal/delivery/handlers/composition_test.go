package handlers

import (
	"time"

	"2026_2_Na_Vse_200/internal/auth"
	"2026_2_Na_Vse_200/internal/config"
	authusecase "2026_2_Na_Vse_200/internal/usecase/auth"
	catalogusecase "2026_2_Na_Vse_200/internal/usecase/catalog"
	"2026_2_Na_Vse_200/internal/usecase/ports"
)

// Existing HTTP regression tests exercise the real application with storage fixtures.
type repositoryTestDeps struct {
	Users    ports.UserRepository
	Sessions ports.SessionRepository
	Catalog  ports.CatalogRepository
	Hasher   ports.PasswordHasher
}

func newRepositoryTestAPI(cfg *config.Config, deps *repositoryTestDeps) *API {
	return New(cfg, &Deps{
		Auth:    authusecase.New(deps.Users, deps.Sessions, deps.Hasher, auth.NewSessionID, time.Now, cfg.SessionTTL),
		Catalog: catalogusecase.New(deps.Catalog),
	})
}
