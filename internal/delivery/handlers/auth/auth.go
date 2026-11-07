package auth

import (
	"encoding/json"
	"fmt"
	"net/http"

	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
	authusecase "2026_2_Na_Vse_200/internal/usecase/auth"
)

const _maxBodySize = 1 << 20

func (a *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, _maxBodySize)
	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(r.Context(), w, http.StatusBadRequest, models.CodeInvalidJSON, models.MsgInvalidJSON)
		return
	}
	result, err := a.usecase.Signup(r.Context(), authusecase.SignupInput{Email: req.Email, Password: req.Password, DisplayName: req.DisplayName})
	if err != nil {
		writeAuthError(r.Context(), w, err)
		return
	}
	a.setSessionCookie(w, string(result.Session.ID))
	response.WriteJSON(r.Context(), w, http.StatusCreated, newUserResponse(result.User))
}

func (a *Handler) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, _maxBodySize)
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(r.Context(), w, http.StatusBadRequest, models.CodeInvalidJSON, models.MsgInvalidJSON)
		return
	}
	result, err := a.usecase.Login(r.Context(), authusecase.LoginInput{Email: req.Email, Password: req.Password})
	if err != nil {
		writeAuthError(r.Context(), w, err)
		return
	}
	a.setSessionCookie(w, string(result.Session.ID))
	response.WriteJSON(r.Context(), w, http.StatusOK, newUserResponse(result.User))
}
func (a *Handler) sessionIDFromRequest(r *http.Request) (string, error) {
	cookie, err := r.Cookie(_sessionCookieName)
	if err != nil {
		return "", fmt.Errorf("read session cookie: %w", err)
	}
	return cookie.Value, nil
}
func (a *Handler) Me(w http.ResponseWriter, r *http.Request) {
	id, err := a.sessionIDFromRequest(r)
	if err != nil {
		writeUnauthorized(r.Context(), w, errNoSessionCookie, err)
		return
	}
	user, err := a.usecase.Me(r.Context(), models.SessionID(id))
	if err != nil {
		writeAuthError(r.Context(), w, err)
		return
	}
	response.WriteJSON(r.Context(), w, http.StatusOK, newUserResponse(user))
}
func (a *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	id, _ := a.sessionIDFromRequest(r)
	if err := a.usecase.Logout(r.Context(), models.SessionID(id)); err != nil {
		writeAuthError(r.Context(), w, err)
		return
	}
	a.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}
