package middleware

import (
	"net/http"

	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/security"
)

// WithCSRF проверяет double-submit cookie на изменяющих запросах.
func WithCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(security.CSRFCookieName)
		cookieToken := ""
		if err == nil {
			cookieToken = cookie.Value
		}
		if err := security.CheckCSRF(cookieToken, r.Header.Get(security.CSRFHeaderName)); err != nil {
			response.WriteError(w, http.StatusForbidden, models.CodeCSRFRejected, models.MsgCSRFRejected)
			return
		}
		next.ServeHTTP(w, r)
	})
}
