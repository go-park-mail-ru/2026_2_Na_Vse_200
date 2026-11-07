package auth

import (
	"net/http"

	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/security"
)

type csrfResponse struct {
	CSRFToken string `json:"csrf_token"`
}

// CSRF выдаёт новый CSRF-токен (cookie + тело ответа) для double-submit.
func (a *Handler) CSRF(w http.ResponseWriter, r *http.Request) {
	token, err := a.issueCSRFCookie(w)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
		return
	}
	response.WriteJSON(w, http.StatusOK, csrfResponse{CSRFToken: token})
}

func (a *Handler) issueCSRFCookie(w http.ResponseWriter) (string, error) {
	token, err := security.NewCSRFToken()
	if err != nil {
		return "", err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     security.CSRFCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(a.cfg.SessionTTL.Seconds()),
		HttpOnly: false,
		Secure:   a.cfg.CookieSecure,
		SameSite: a.sessionSameSite(),
	})
	return token, nil
}

func (a *Handler) clearCSRFCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     security.CSRFCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: false,
		Secure:   a.cfg.CookieSecure,
		SameSite: a.sessionSameSite(),
	})
}
