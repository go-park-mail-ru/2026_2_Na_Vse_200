package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/apimessage"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/pkg/response"
)

const _sessionKey ctxKey = 1

func WithAuth(cookieName string, sessions repository.SessionRepositoryInterface) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(cookieName)
			if err != nil {
				response.WriteError(w, http.StatusUnauthorized,
					apimessage.CodeUnauthorized, apimessage.MsgUnauthorized)
				return
			}

			session, err := sessions.GetByID(r.Context(), models.SessionID(cookie.Value))
			switch {
			case errors.Is(err, repository.ErrSessionNotFound), errors.Is(err, repository.ErrSessionExpired):
				response.WriteError(w, http.StatusUnauthorized,
					apimessage.CodeUnauthorized, apimessage.MsgUnauthorized)
				return
			case err != nil:
				log.Printf("auth middleware: чтение сессии: %v", err)
				response.WriteError(w, http.StatusInternalServerError,
					apimessage.CodeInternal, apimessage.MsgInternal)
				return
			}

			ctx := context.WithValue(r.Context(), _sessionKey, session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func SessionFromContext(ctx context.Context) (models.Session, bool) {
	session, ok := ctx.Value(_sessionKey).(models.Session)
	return session, ok
}
