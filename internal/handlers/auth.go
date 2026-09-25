package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/storage"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/validation"
)

// maxBodySize ограничивает размер тела запроса. Без ограничения клиент может
// прислать гигабайтный JSON и занять память сервера.
const maxBodySize = 1 << 20 // 1 МБ

// signupRequest — тело запроса регистрации.
//
// Отдельный тип, а не models.User: так в модель не попадут лишние поля
// из запроса, а клиент не сможет подсунуть, например, чужой ID.
type signupRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

// userResponse — безопасное представление аккаунта для клиента.
//
// Отдельный тип нужен прежде всего ради того, чего здесь НЕТ: поля с хешем
// пароля. Физически невозможно отдать наружу то, чего нет в структуре.
//
// ID — строка: тип ключа в БД ещё не зафиксирован, а в JavaScript целые
// числа больше 2^53 теряют точность.
type userResponse struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

// newUserResponse переводит доменную модель в ответ API.
func newUserResponse(user models.User) userResponse {
	resp := userResponse{
		ID:          strconv.FormatInt(int64(user.ID), 10),
		Email:       user.Email,
		DisplayName: user.DisplayName,
	}
	// В контракте avatar_url — строка или null, а не пустая строка.
	if user.AvatarURL != "" {
		url := user.AvatarURL
		resp.AvatarURL = &url
	}
	return resp
}

// Signup создаёт аккаунт.
//
// POST /api/v1/auth/signup, см. docs/api.md, раздел 6.1.
// Автоматического входа нет: после успеха фронтенд ведёт пользователя
// на форму входа, cookie здесь не выставляется.
func (a *API) Signup(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Ошибку разбора проверяем обязательно: без этого поля молча
		// остались бы пустыми, и мы бы завели аккаунт из мусора.
		writeError(w, http.StatusBadRequest, CodeInvalidJSON, "Некорректный JSON в теле запроса")
		return
	}

	checked := validation.Signup(validation.SignupInput{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if !checked.Valid() {
		writeValidationError(w, checked.Fields)
		return
	}

	hash, err := a.deps.Hasher.Hash(checked.Password)
	if err != nil {
		// В лог уходит причина, клиенту — общий текст. Сам пароль
		// не логируется ни при каких обстоятельствах.
		writeInternalError(w, "хеширование пароля", err)
		return
	}

	user, err := a.deps.Users.Create(r.Context(), models.User{
		Email:        checked.Email,
		PasswordHash: hash,
		DisplayName:  checked.DisplayName,
	})
	switch {
	case errors.Is(err, storage.ErrEmailTaken):
		// Занятый email определяет само хранилище при вставке, а не отдельная
		// проверка «есть ли такой»: между проверкой и вставкой успел бы
		// вклиниться второй такой же запрос.
		writeError(w, http.StatusConflict, CodeEmailTaken, "Пользователь с таким email уже существует")
		return
	case err != nil:
		writeInternalError(w, "создание пользователя", err)
		return
	}

	writeJSON(w, http.StatusCreated, newUserResponse(user))
}
