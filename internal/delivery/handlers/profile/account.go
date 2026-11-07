package profile

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"2026_2_Na_Vse_200/internal/config"
	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
)

type AccountUsecase interface {
	ChangeEmail(context.Context, models.UserID, models.UserID, string, string) (models.User, error)
	ChangePassword(context.Context, models.UserID, models.UserID, string, string) error
	Delete(context.Context, models.UserID, models.UserID, string) error
}

type AccountHandler struct {
	usecase AccountUsecase
	cfg     *config.Config
}

func NewAccount(usecase AccountUsecase, cfg *config.Config) *AccountHandler {
	return &AccountHandler{usecase: usecase, cfg: cfg}
}

type accountRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
	Email           string `json:"email"`
}

func (h *AccountHandler) request(w http.ResponseWriter, r *http.Request) (models.UserID, models.UserID, accountRequest, bool) {
	var req accountRequest
	session, ok := middleware.SessionFromContext(r.Context())
	if !ok {
		response.WriteError(w, 401, models.CodeUnauthorized, models.MsgUnauthorized)
		return "", "", req, false
	}
	id, ok := parseUserID(w, r)
	if !ok {
		return "", "", req, false
	}
	if id != session.UserID {
		response.WriteError(w, 403, models.CodeForbidden, models.MsgForbidden)
		return "", "", req, false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		response.WriteError(w, 400, models.CodeInvalidJSON, models.MsgInvalidJSON)
		return "", "", req, false
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		response.WriteError(w, 400, models.CodeInvalidJSON, models.MsgInvalidJSON)
		return "", "", req, false
	}
	return session.UserID, id, req, true
}

func writeAccountError(w http.ResponseWriter, err error) bool {
	if errors.Is(err, models.ErrCurrentPassword) {
		response.WriteFieldsError(w, 400, "invalid_current_password", "Неверный текущий пароль", map[string]string{"current_password": "Неверный текущий пароль"})
		return false
	}
	return writeProfileError(w, err, "изменение аккаунта")
}

func (h *AccountHandler) ChangeEmail(w http.ResponseWriter, r *http.Request) {
	owner, id, req, ok := h.request(w, r)
	if !ok {
		return
	}
	user, err := h.usecase.ChangeEmail(r.Context(), owner, id, req.CurrentPassword, req.Email)
	if !writeAccountError(w, err) {
		return
	}
	body, err := newProfileResponse(user)
	if !writeAccountError(w, err) {
		return
	}
	response.WriteJSON(w, 200, body)
}

func (h *AccountHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	owner, id, req, ok := h.request(w, r)
	if !ok {
		return
	}
	if !writeAccountError(w, h.usecase.ChangePassword(r.Context(), owner, id, req.CurrentPassword, req.NewPassword)) {
		return
	}
	h.clearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AccountHandler) Delete(w http.ResponseWriter, r *http.Request) {
	owner, id, req, ok := h.request(w, r)
	if !ok {
		return
	}
	if !writeAccountError(w, h.usecase.Delete(r.Context(), owner, id, req.CurrentPassword)) {
		return
	}
	h.clearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AccountHandler) clearCookie(w http.ResponseWriter) {
	sameSite := http.SameSiteLaxMode
	if h.cfg.AllowedOrigin != "" && h.cfg.CookieSecure {
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{Name: "session_id", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: h.cfg.CookieSecure, SameSite: sameSite})
}
