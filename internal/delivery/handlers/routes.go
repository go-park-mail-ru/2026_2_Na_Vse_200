package handlers

import (
	"2026_2_Na_Vse_200/internal/delivery/handlers/favorites"
	"net/http"

	"2026_2_Na_Vse_200/internal/config"
	"2026_2_Na_Vse_200/internal/delivery/handlers/artist"
	"2026_2_Na_Vse_200/internal/delivery/handlers/auth"
	"2026_2_Na_Vse_200/internal/delivery/handlers/health"
	"2026_2_Na_Vse_200/internal/delivery/handlers/home"
	"2026_2_Na_Vse_200/internal/delivery/handlers/media"
	"2026_2_Na_Vse_200/internal/delivery/handlers/profile"
	"2026_2_Na_Vse_200/internal/delivery/middleware"
)

const sessionCookieName = "session_id"

type Deps struct {
	Favorites favorites.Repository
	Auth      auth.Usecase
	Catalog   home.CatalogUsecase
	Profile   profile.ProfileUsecase
	Account   profile.AccountUsecase
	Artist    artist.ArtistUsecase
	Media     *media.MediaHandler
}

type API struct {
	favorites      *favorites.Handler
	auth           auth.Usecase
	authH          *auth.Handler
	home           *home.Handler
	profile        *profile.ProfileHandler
	account        *profile.AccountHandler
	accountEnabled bool
	artist         *artist.ArtistHandler
	media          *media.MediaHandler
}

func New(cfg *config.Config, deps *Deps) *API {
	api := &API{
		auth:           deps.Auth,
		authH:          auth.New(cfg, deps.Auth),
		home:           home.New(deps.Catalog),
		profile:        profile.NewProfileHandler(deps.Profile),
		account:        profile.NewAccountHandler(deps.Account, cfg),
		accountEnabled: deps.Account != nil,
		artist:         artist.NewArtistHandler(deps.Artist),
		media:          deps.Media,
	}
	if deps.Favorites != nil {
		api.favorites = favorites.NewFavoritesHandler(deps.Favorites)
	}
	return api
}

func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health.Health)
	mux.HandleFunc("GET /api/v1/auth/csrf", a.authH.CSRF)
	mux.HandleFunc("POST /api/v1/auth/signup", a.authH.Signup)
	mux.HandleFunc("POST /api/v1/auth/login", a.authH.Login)
	mux.HandleFunc("GET /api/v1/auth/me", a.authH.Me)
	mux.HandleFunc("POST /api/v1/auth/logout", a.authH.Logout)
	mux.HandleFunc("GET /api/v1/home", a.home.Home)

	if a.auth != nil && a.profile != nil {
		authUser := middleware.WithAuth(sessionCookieName, a.auth.SessionByID)
		if a.accountEnabled {
			mux.Handle("PUT /api/v1/users/{id}/email", authUser(http.HandlerFunc(a.account.ChangeEmail)))
			mux.Handle("PUT /api/v1/users/{id}/password", authUser(http.HandlerFunc(a.account.ChangePassword)))
			mux.Handle("DELETE /api/v1/users/{id}", authUser(http.HandlerFunc(a.account.Delete)))
		}
		mux.Handle("GET /api/v1/users/{id}", authUser(http.HandlerFunc(a.profile.Get)))
		mux.Handle("PUT /api/v1/users/{id}", authUser(http.HandlerFunc(a.profile.Update)))
		mux.Handle("POST /api/v1/users/{id}/avatar", authUser(http.HandlerFunc(a.profile.UploadAvatar)))
	}
	if a.auth != nil && a.favorites != nil {
		authUser := middleware.WithAuth(sessionCookieName, a.auth.SessionByID)
		mux.Handle("GET /api/v1/favorites", authUser(http.HandlerFunc(a.favorites.List)))
		mux.Handle("PUT /api/v1/tracks/{id}/favorite", authUser(http.HandlerFunc(a.favorites.Set)))
		mux.Handle("DELETE /api/v1/tracks/{id}/favorite", authUser(http.HandlerFunc(a.favorites.Set)))
	}
	if a.artist != nil {
		mux.HandleFunc("GET /api/v1/artists/{id}", a.artist.Get)
	}
	if a.media != nil {
		mux.HandleFunc("GET /api/v1/media/{id}", a.media.Get)
		mux.HandleFunc("GET /api/v1/tracks/{id}/audio", a.media.Audio)
	}
	return mux
}
