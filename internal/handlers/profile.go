package handlers

import (
	"errors"
	"net/http"
	"time"
	"uuid"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/apimessage"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/pkg/response"
)

type profileResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url"`
	CreatedAt   time.Time `json:"created_at"`
}

func newProfileResponse(user models.User) profileResponse {
	resp := profileResponse{
		ID:          string(user.ID),
		Email:       user.Email,
		DisplayName: user.DisplayName,
		CreatedAt:   user.CreatedAt.UTC(),
	}
	if user.AvatarURL != "" {
		avatar := user.AvatarURL
		resp.AvatarURL = &avatar
	}
	return resp
}

func (a *API) Profile(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.SessionFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, errNoSessionInContext, nil)
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.WriteError(w, http.StatusNotFound, apimessage.CodeNotFound, apimessage.MsgNotFound)
		return
	}

	if models.UserID(id.String()) != session.UserID {
		writeForbidden(w, errNotOwner)
		return
	}

	user, err := a.deps.Users.GetByID(r.Context(), session.UserID)
	switch {
	case errors.Is(err, repository.ErrUserNotFound):
		writeUnauthorized(w, errUserGone, err)
		return
	case err != nil:
		writeInternalError(w, errFindUser, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, newProfileResponse(user))
}
