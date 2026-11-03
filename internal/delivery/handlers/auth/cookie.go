package auth

import "net/http"

const _sessionCookieName = "session_id"

func (a *Handler) setSessionCookie(w http.ResponseWriter, id string) {
	http.SetCookie(w, &http.Cookie{
		Name:     _sessionCookieName,
		Value:    id,
		Path:     "/",
		MaxAge:   int(a.cfg.SessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   a.cfg.CookieSecure,
		SameSite: a.sessionSameSite(),
	})
}

// Параметры удаляемой cookie должны совпадать с параметрами установленной.
func (a *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     _sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   a.cfg.CookieSecure,
		SameSite: a.sessionSameSite(),
	})
}

// SameSite=None используется для запросов между сайтами и требует Secure.
func (a *Handler) sessionSameSite() http.SameSite {
	if a.cfg.AllowedOrigin != "" && a.cfg.CookieSecure {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}
