package handlers

import "net/http"

const sessionCookieName = "session_id"

// setSessionCookie кладёт идентификатор сессии в cookie.
func (a *API) setSessionCookie(w http.ResponseWriter, id string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    id,
		Path:     "/",
		MaxAge:   int(a.cfg.SessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   a.cfg.CookieSecure,
		SameSite: a.sessionSameSite(),
	})
}

// sessionSameSite выбирает режим по тому, живут ли фронтенд и API на одном
// адресе: при разных cookie доедет только с None, а его браузер принимает
// лишь вместе с Secure.
func (a *API) sessionSameSite() http.SameSite {
	if a.cfg.AllowedOrigin == "" {
		return http.SameSiteLaxMode
	}
	return http.SameSiteNoneMode
}

// clearSessionCookie гасит cookie. MaxAge < 0 уходит браузеру как Max-Age=0.
func (a *API) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   a.cfg.CookieSecure,
		SameSite: a.sessionSameSite(),
	})
}
