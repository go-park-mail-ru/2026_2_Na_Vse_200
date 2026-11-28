package handlers

import (
	"net/http"

	"2026_2_Na_Vse_200/internal/config"
	"2026_2_Na_Vse_200/internal/delivery/handlers/artist"
	"2026_2_Na_Vse_200/internal/delivery/handlers/auth"
	"2026_2_Na_Vse_200/internal/delivery/handlers/favorites"
	"2026_2_Na_Vse_200/internal/delivery/handlers/health"
	"2026_2_Na_Vse_200/internal/delivery/handlers/home"
	"2026_2_Na_Vse_200/internal/delivery/handlers/media"
	"2026_2_Na_Vse_200/internal/delivery/handlers/music"
	"2026_2_Na_Vse_200/internal/delivery/handlers/onlineconfadmin"
	"2026_2_Na_Vse_200/internal/delivery/handlers/playlist"
	"2026_2_Na_Vse_200/internal/delivery/handlers/profile"
	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"2026_2_Na_Vse_200/internal/onlineconf"
)

const sessionCookieName = "session_id"

type Deps struct {
	Music              music.Repository
	Playlists          *playlist.Handler
	Favorites          favorites.Repository
	Auth               auth.Usecase
	Catalog            home.CatalogUsecase
	Profile            profile.ProfileUsecase
	Account            profile.AccountUsecase
	Artist             artist.ArtistUsecase
	Media              *media.MediaHandler
	CheckProfileAccess profile.CheckAccess
	OnlineConf         *onlineconf.Cache
	OnlineConfAdmin    *onlineconfadmin.Handler
}

type API struct {
	favorites       *favorites.Handler
	music           *music.Handler
	playlists       *playlist.Handler
	auth            auth.Usecase
	authH           *auth.Handler
	home            *home.Handler
	profile         *profile.ProfileHandler
	account         *profile.AccountHandler
	accountEnabled  bool
	artist          *artist.ArtistHandler
	media           *media.MediaHandler
	onlineConfAdmin *onlineconfadmin.Handler
}

func New(cfg *config.Config, deps *Deps) *API {
	var sessionByID middleware.SessionByIDFunc
	if deps.Auth != nil {
		sessionByID = deps.Auth.SessionByID
	}
	api := &API{
		auth:            deps.Auth,
		authH:           auth.New(cfg, deps.Auth),
		home:            home.New(deps.Catalog, deps.OnlineConf, sessionCookieName, sessionByID),
		profile:         profile.NewProfileHandler(deps.Profile, deps.CheckProfileAccess),
		account:         profile.NewAccountHandler(deps.Account, cfg, deps.CheckProfileAccess),
		accountEnabled:  deps.Account != nil,
		artist:          artist.NewArtistHandler(deps.Artist),
		media:           deps.Media,
		playlists:       deps.Playlists,
		onlineConfAdmin: deps.OnlineConfAdmin,
	}
	if deps.Music != nil {
		api.music = music.New(deps.Music)
	}
	if deps.Favorites != nil {
		api.favorites = favorites.NewFavoritesHandler(deps.Favorites)
	}
	return api
}

func (a *API) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health.Health)
	mux.HandleFunc("GET /api/v1/auth/csrf", a.authH.CSRF)
	mux.HandleFunc("POST /api/v1/auth/signup", a.authH.Signup)
	mux.HandleFunc("POST /api/v1/auth/login", a.authH.Login)
	mux.HandleFunc("GET /api/v1/auth/me", a.authH.Me)
	mux.HandleFunc("POST /api/v1/auth/logout", a.authH.Logout)
	mux.HandleFunc("GET /api/v1/home", a.home.Home)
	if a.auth != nil && a.onlineConfAdmin != nil {
		a.onlineConfAdmin.Register(mux, middleware.WithAuth(sessionCookieName, a.auth.SessionByID))
	}

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
	if a.music != nil {
		mux.HandleFunc("GET /api/v1/search", a.music.Search)
		mux.HandleFunc("GET /api/v1/albums/{id}", a.music.Album)
	}
	if a.auth != nil && a.playlists != nil {
		a.playlists.Register(mux, middleware.WithAuth(sessionCookieName, a.auth.SessionByID))
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
