package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/apimessage"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/auth"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/pkg/response"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/pkg/validation"
)

const maxBodySize = 1 << 20 // 1 МБ

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
		ID:          strconv.FormatInt(int64(user.ID), 10),
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

// signIn заводит сессию для user и кладёт её идентификатор в cookie.
func (a *API) signIn(ctx context.Context, w http.ResponseWriter, user models.User) error {
	session := models.Session{
		ID:        auth.NewSessionID(),
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(a.cfg.SessionTTL),
	}
	if err := a.deps.Sessions.Create(ctx, session); err != nil {
		return fmt.Errorf("ошибка сохранения сессии: %w", err)
	}

	a.setSessionCookie(w, session.ID)
	return nil
}

// Signup создаёт аккаунт и сразу выполняет вход: POST /api/v1/auth/signup.
func (a *API) Signup(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, apimessage.CodeInvalidJSON, apimessage.MsgInvalidJSON)
		return
	}

	form := validation.Signup(validation.SignupInput{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if !form.Valid() {
		response.WriteFieldsError(w, http.StatusBadRequest,
			apimessage.CodeValidationFailed, apimessage.MsgValidationFailed, form.Fields)
		return
	}

	hash, err := a.deps.Hasher.Hash(form.Password)
	if err != nil {
		writeInternalError(w, "ошибка хеширования пароля", err)
		return
	}

	user, err := a.deps.Users.Create(r.Context(), models.User{
		Email:        form.Email,
		PasswordHash: hash,
		DisplayName:  form.DisplayName,
	})
	switch {
	case errors.Is(err, repository.ErrEmailTaken):
		response.WriteError(w, http.StatusConflict, apimessage.CodeEmailTaken, apimessage.MsgEmailTaken)
		return
	case err != nil:
		writeInternalError(w, "ошибка создания пользователя", err)
		return
	}

	if err := a.signIn(r.Context(), w, user); err != nil {
		log.Printf("ошибка автовхода после регистрации: %v", err)
	}

	response.WriteJSON(w, http.StatusCreated, newUserResponse(user))
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login выдаёт сессию по email и паролю: POST /api/v1/auth/login.
func (a *API) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, apimessage.CodeInvalidJSON, apimessage.MsgInvalidJSON)
		return
	}

	form := validation.Login(validation.LoginInput{Email: req.Email, Password: req.Password})
	if !form.Valid() {
		response.WriteFieldsError(w, http.StatusBadRequest,
			apimessage.CodeValidationFailed, apimessage.MsgValidationFailed, form.Fields)
		return
	}

	user, err := a.deps.Users.GetByEmail(r.Context(), form.Email)
	switch {
	case errors.Is(err, repository.ErrUserNotFound):
		writeInvalidCredentials(w)
		return
	case err != nil:
		writeInternalError(w, "ошибка поиска пользователя", err)
		return
	}

	matched, err := a.deps.Hasher.Verify(form.Password, user.PasswordHash)
	if err != nil {
		writeInternalError(w, "ошибка проверки пароля", err)
		return
	}
	if !matched {
		writeInvalidCredentials(w)
		return
	}

	if err := a.signIn(r.Context(), w, user); err != nil {
		writeInternalError(w, "ошибка входа пользователя", err)
		return
	}

	response.WriteJSON(w, http.StatusOK, newUserResponse(user))
}

// writeInvalidCredentials отвечает одинаково на неизвестный email и на неверный
// пароль: по разнице ответов перебирают зарегистрированные адреса.
func writeInvalidCredentials(w http.ResponseWriter) {
	response.WriteError(w, http.StatusUnauthorized,
		apimessage.CodeInvalidCredentials, apimessage.MsgInvalidCredentials)
}

// sessionIDFromRequest достаёт идентификатор сессии из cookie.
// Пустая строка означает, что клиент её не прислал.
func (a *API) sessionIDFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// Me отдаёт пользователя текущей сессии: GET /api/v1/auth/me.
// Фронтенд зовёт её при запуске, чтобы восстановиться после перезагрузки страницы.
func (a *API) Me(w http.ResponseWriter, r *http.Request) {
	id := a.sessionIDFromRequest(r)
	if id == "" {
		writeUnauthorized(w)
		return
	}

	session, err := a.deps.Sessions.GetByID(r.Context(), id)
	switch {
	case errors.Is(err, repository.ErrSessionNotFound), errors.Is(err, repository.ErrSessionExpired):
		// Неизвестная и истёкшая сессии отвечают одинаково: разница подсказывала бы,
		// что идентификатор угадан верно.
		writeUnauthorized(w)
		return
	case err != nil:
		// Сбой хранилища под 401 маскировать нельзя: фронтенд принял бы падение
		// базы за разлогин и молча показал гостевой интерфейс.
		writeInternalError(w, "ошибка чтения сессии", err)
		return
	}

	user, err := a.deps.Users.GetByID(r.Context(), session.UserID)
	switch {
	case errors.Is(err, repository.ErrUserNotFound):
		writeUnauthorized(w)
		return
	case err != nil:
		writeInternalError(w, "ошибка поиска пользователя", err)
		return
	}

	response.WriteJSON(w, http.StatusOK, newUserResponse(user))
}

// Logout завершает сессию: POST /api/v1/auth/logout.
// Выход без cookie и повторный выход тоже считаются успехом — результат тот же.
func (a *API) Logout(w http.ResponseWriter, r *http.Request) {
	if id := a.sessionIDFromRequest(r); id != "" {
		// Удаляем именно на сервере: погасить одну cookie мало,
		// украденный идентификатор остался бы рабочим.
		if err := a.deps.Sessions.Delete(r.Context(), id); err != nil {
			writeInternalError(w, "ошибка удаления сессии", err)
			return
		}
	}

	a.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// writeUnauthorized отвечает одинаково на отсутствующую, неизвестную
// и истёкшую сессию.
func writeUnauthorized(w http.ResponseWriter) {
	response.WriteError(w, http.StatusUnauthorized,
		apimessage.CodeUnauthorized, apimessage.MsgUnauthorized)
}
