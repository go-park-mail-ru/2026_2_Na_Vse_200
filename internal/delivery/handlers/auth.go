package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"2026_2_Na_Vse_200/internal/apimessage"
	"2026_2_Na_Vse_200/internal/models"
	authusecase "2026_2_Na_Vse_200/internal/usecase/auth"
	"2026_2_Na_Vse_200/pkg/response"
)

const _maxBodySize = 1 << 20 // 1 МБ

type signupRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

// userResponse — представление аккаунта для клиента: без хеша пароля,
// с идентификатором в виде строки.
type userResponse struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

func newUserResponse(user models.User) userResponse {
	resp := userResponse{
		ID:          string(user.ID),
		Email:       user.Email,
		DisplayName: user.DisplayName,
	}
	// В контракте avatar_url — строка или null.
	if user.AvatarURL != "" {
		avatar := user.AvatarURL
		resp.AvatarURL = &avatar
	}
	return resp
}

// Signup adapts HTTP input to the registration scenario.
func (a *API) Signup(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, _maxBodySize)
	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, apimessage.CodeInvalidJSON, apimessage.MsgInvalidJSON)
		return
	}
	result, err := a.deps.Auth.Signup(r.Context(), authusecase.SignupInput{Email: req.Email, Password: req.Password, DisplayName: req.DisplayName})
	if err != nil {
		writeAuthError(w, err)
		return
	}
	a.setSessionCookie(w, string(result.Session.ID))
	response.WriteJSON(w, http.StatusCreated, newUserResponse(result.User))
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *API) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, _maxBodySize)
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, apimessage.CodeInvalidJSON, apimessage.MsgInvalidJSON)
		return
	}
	result, err := a.deps.Auth.Login(r.Context(), authusecase.LoginInput{Email: req.Email, Password: req.Password})
	if err != nil {
		writeAuthError(w, err)
		return
	}
	a.setSessionCookie(w, string(result.Session.ID))
	response.WriteJSON(w, http.StatusOK, newUserResponse(result.User))
}
func (a *API) sessionIDFromRequest(r *http.Request) (string, error) {
	cookie, err := r.Cookie(_sessionCookieName)
	if err != nil {
		return "", fmt.Errorf("read session cookie: %w", err)
	}
	return cookie.Value, nil
}
func (a *API) Me(w http.ResponseWriter, r *http.Request) {
	id, err := a.sessionIDFromRequest(r)
	if err != nil {
		writeUnauthorized(w, errNoSessionCookie, err)
		return
	}
	user, err := a.deps.Auth.Me(r.Context(), models.SessionID(id))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, newUserResponse(user))
}
func (a *API) Logout(w http.ResponseWriter, r *http.Request) {
	id, _ := a.sessionIDFromRequest(r)
	if err := a.deps.Auth.Logout(r.Context(), models.SessionID(id)); err != nil {
		writeAuthError(w, err)
		return
	}
	a.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}
func writeAuthError(w http.ResponseWriter, err error) {
	var validationErr *authusecase.ValidationError
	switch {
	case errors.As(err, &validationErr):
		response.WriteFieldsError(w, http.StatusBadRequest, apimessage.CodeValidationFailed, apimessage.MsgValidationFailed, validationErr.Fields)
	case errors.Is(err, models.ErrEmailTaken):
		response.WriteError(w, http.StatusConflict, apimessage.CodeEmailTaken, apimessage.MsgEmailTaken)
	case errors.Is(err, authusecase.ErrInvalidCredentials):
		writeInvalidCredentials(w, err.Error())
	case errors.Is(err, authusecase.ErrUnauthorized):
		writeUnauthorized(w, errSessionRejected, err)
	default:
		writeInternalError(w, "authentication failed", err)
	}
}
