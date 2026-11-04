package profile

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"uuid"

	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
	profileusecase "2026_2_Na_Vse_200/internal/usecase/profile"
)

const (
	maxBodySize   = 1_048_576
	maxAvatarForm = profileusecase.MaxAvatarBytes + 1_048_576
	avatarField   = "avatar"
)

type ProfileUsecase interface {
	Get(ctx context.Context, sessionUserID, requestedID models.UserID) (models.User, error)
	Update(ctx context.Context, sessionUserID, requestedID models.UserID, in profileusecase.UpdateInput) (models.User, error)
	UploadAvatar(ctx context.Context, sessionUserID, requestedID models.UserID, mimeType string, size int64, body io.Reader) (models.User, error)
}

type ProfileHandler struct {
	usecase ProfileUsecase
}

func NewProfileHandler(usecase ProfileUsecase) *ProfileHandler {
	return &ProfileHandler{usecase: usecase}
}

func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	session, requestedID, ok := ownerRequest(w, r)
	if !ok {
		return
	}

	user, err := h.usecase.Get(r.Context(), session.UserID, requestedID)
	if !writeProfileError(w, err, "чтение профиля") {
		return
	}
	body, err := newProfileResponse(user)
	if err != nil {
		log.Printf("сборка ответа профиля: %v", err)
		response.WriteError(w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
		return
	}
	response.WriteJSON(w, http.StatusOK, body)
}

func (h *ProfileHandler) Update(w http.ResponseWriter, r *http.Request) {
	session, requestedID, ok := ownerRequest(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, models.CodeInvalidJSON, models.MsgInvalidJSON)
		return
	}

	user, err := h.usecase.Update(r.Context(), session.UserID, requestedID, profileusecase.UpdateInput{
		Email:       req.Email,
		DisplayName: req.DisplayName,
	})
	if !writeProfileError(w, err, "обновление профиля") {
		return
	}
	body, err := newProfileResponse(user)
	if err != nil {
		log.Printf("сборка ответа профиля: %v", err)
		response.WriteError(w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
		return
	}
	response.WriteJSON(w, http.StatusOK, body)
}

func (h *ProfileHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	session, requestedID, ok := ownerRequest(w, r)
	if !ok {
		return
	}

	if err := r.ParseMultipartForm(maxAvatarForm); err != nil {
		response.WriteError(w, http.StatusBadRequest, models.CodeInvalidJSON, models.MsgInvalidJSON)
		return
	}
	file, _, err := r.FormFile(avatarField)
	if err != nil {
		response.WriteFieldsError(w, http.StatusBadRequest, models.CodeValidationFailed, models.MsgValidationFailed, map[string]string{
			avatarField: "Добавьте файл аватара",
		})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, profileusecase.MaxAvatarBytes+1))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, models.CodeInvalidJSON, models.MsgInvalidJSON)
		return
	}
	mimeType, err := profileusecase.DetectAvatarMIME(data)
	if err != nil || len(data) == 0 || int64(len(data)) > profileusecase.MaxAvatarBytes {
		response.WriteFieldsError(w, http.StatusBadRequest, models.CodeValidationFailed, models.MsgValidationFailed, map[string]string{
			avatarField: "Нужен JPEG, PNG или WebP до 2 МБ",
		})
		return
	}

	user, err := h.usecase.UploadAvatar(
		r.Context(),
		session.UserID,
		requestedID,
		mimeType,
		int64(len(data)),
		bytes.NewReader(data),
	)
	if !writeProfileError(w, err, "загрузка аватара") {
		return
	}
	body, err := newProfileResponse(user)
	if err != nil {
		log.Printf("сборка ответа профиля: %v", err)
		response.WriteError(w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
		return
	}
	response.WriteJSON(w, http.StatusOK, body)
}

type updateRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

func ownerRequest(w http.ResponseWriter, r *http.Request) (models.Session, models.UserID, bool) {
	session, ok := middleware.SessionFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, models.CodeUnauthorized, models.MsgUnauthorized)
		return models.Session{}, "", false
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.WriteError(w, http.StatusNotFound, models.CodeNotFound, models.MsgNotFound)
		return models.Session{}, "", false
	}
	return session, models.UserID(id.String()), true
}

func writeProfileError(w http.ResponseWriter, err error, cause string) bool {
	if err == nil {
		return true
	}
	var validationErr *profileusecase.ValidationError
	switch {
	case errors.As(err, &validationErr):
		response.WriteFieldsError(w, http.StatusBadRequest, models.CodeValidationFailed, models.MsgValidationFailed, validationErr.Fields)
	case errors.Is(err, profileusecase.ErrForbidden):
		response.WriteError(w, http.StatusForbidden, models.CodeForbidden, models.MsgForbidden)
	case errors.Is(err, profileusecase.ErrUnauthorized):
		response.WriteError(w, http.StatusUnauthorized, models.CodeUnauthorized, models.MsgUnauthorized)
	case errors.Is(err, models.ErrEmailTaken):
		response.WriteError(w, http.StatusConflict, models.CodeEmailTaken, models.MsgEmailTaken)
	case errors.Is(err, profileusecase.ErrInvalidAvatar):
		response.WriteFieldsError(w, http.StatusBadRequest, models.CodeValidationFailed, models.MsgValidationFailed, map[string]string{
			avatarField: "Нужен JPEG, PNG или WebP до 2 МБ",
		})
	default:
		log.Printf("%s: %v", cause, err)
		response.WriteError(w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
	}
	return false
}
