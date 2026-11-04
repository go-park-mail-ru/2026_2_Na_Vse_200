package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"

	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
)

type sessionContextKey struct{}

func WithAuth(cookieName string, sessions ports.SessionRepository) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(cookieName)
			if err != nil {
				response.WriteError(w, http.StatusUnauthorized, models.CodeUnauthorized, models.MsgUnauthorized)
				return
			}

			session, err := sessions.GetByID(r.Context(), models.SessionID(cookie.Value))
			switch {
			case errors.Is(err, models.ErrSessionNotFound), errors.Is(err, models.ErrSessionExpired):
				response.WriteError(w, http.StatusUnauthorized, models.CodeUnauthorized, models.MsgUnauthorized)
				return
			case err != nil:
				log.Printf("auth middleware: чтение сессии: %v", err)
				response.WriteError(w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
				return
			}

			ctx := context.WithValue(r.Context(), sessionContextKey{}, session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func SessionFromContext(ctx context.Context) (models.Session, bool) {
	session, ok := ctx.Value(sessionContextKey{}).(models.Session)
	return session, ok
}
