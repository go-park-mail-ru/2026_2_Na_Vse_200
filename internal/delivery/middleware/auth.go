package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/rs/zerolog"

	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
)

type sessionContextKey struct{}

// SessionByIDFunc получает сессию через usecase авторизации.
type SessionByIDFunc func(ctx context.Context, id models.SessionID) (models.Session, error)

func WithAuth(cookieName string, sessionByID SessionByIDFunc) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(cookieName)
			if err != nil {
				response.WriteError(r.Context(), w, http.StatusUnauthorized, models.CodeUnauthorized, models.MsgUnauthorized)
				return
			}

			session, err := sessionByID(r.Context(), models.SessionID(cookie.Value))
			switch {
			case errors.Is(err, models.ErrSessionNotFound), errors.Is(err, models.ErrSessionExpired):
				response.WriteError(r.Context(), w, http.StatusUnauthorized, models.CodeUnauthorized, models.MsgUnauthorized)
				return
			case err != nil:
				zerolog.Ctx(r.Context()).Error().Err(err).Str("operation", "auth.session").Msg("чтение сессии")
				response.WriteError(r.Context(), w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
				return
			}

			logger := zerolog.Ctx(r.Context()).With().Str("user_id", string(session.UserID)).Logger()
			ctx := context.WithValue(r.Context(), sessionContextKey{}, session)
			ctx = logger.WithContext(ctx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func SessionFromContext(ctx context.Context) (models.Session, bool) {
	session, ok := ctx.Value(sessionContextKey{}).(models.Session)
	return session, ok
}
