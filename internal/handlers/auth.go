package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/storage"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/pkg/apimessage"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/pkg/response"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/pkg/validation"
)

const maxBodySize = 1 << 20 // 1 МБ

type signupRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

// userResponse — представление аккаунта для клиента. Поля с хешем пароля здесь
// нет намеренно: отдать наружу то, чего нет в структуре, невозможно.
// ID отдаётся строкой, как требует контракт.
type userResponse struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

func newUserResponse(user models.User) userResponse {
	resp := userResponse{
		ID:          strconv.FormatInt(int64(user.ID), 10),
		Email:       user.Email,
		DisplayName: user.DisplayName,
	}
	// В контракте avatar_url — строка или null.
	if user.AvatarURL != "" {
		url := user.AvatarURL
		resp.AvatarURL = &url
	}
	return resp
}

// Signup создаёт аккаунт: POST /api/v1/auth/signup.
// Cookie не выставляется — автоматического входа после регистрации нет.
func (a *API) Signup(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, apimessage.CodeInvalidJSON, apimessage.MsgInvalidJSON)
		return
	}

	checked := validation.Signup(validation.SignupInput{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if !checked.Valid() {
		response.WriteValidationError(w, checked.Fields)
		return
	}

	hash, err := a.deps.Hasher.Hash(checked.Password)
	if err != nil {
		response.WriteInternalError(w, "хеширование пароля", err)
		return
	}

	user, err := a.deps.Users.Create(r.Context(), models.User{
		Email:        checked.Email,
		PasswordHash: hash,
		DisplayName:  checked.DisplayName,
	})
	switch {
	case errors.Is(err, storage.ErrEmailTaken):
		response.WriteError(w, http.StatusConflict, apimessage.CodeEmailTaken, apimessage.MsgEmailTaken)
		return
	case err != nil:
		response.WriteInternalError(w, "создание пользователя", err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, newUserResponse(user))
}
