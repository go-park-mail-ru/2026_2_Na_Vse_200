package auth

import (
	authusecase "2026_2_Na_Vse_200/internal/usecase/auth"
	"errors"
	"log"
	"net/http"

	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
)

const (
	errNoSessionCookie = "запрос без cookie сессии"
	errSessionRejected = "сессия неизвестна или истекла"
)

func logCause(cause string, err error) {
	if err != nil {
		log.Printf("%s: %v", cause, err)
		return
	}
	log.Print(cause)
}

func writeInternalError(w http.ResponseWriter, cause string, err error) {
	logCause(cause, err)
	response.WriteError(w, http.StatusInternalServerError,
		models.CodeInternal, models.MsgInternal)
}

// Отсутствующая, неизвестная и истёкшая сессии дают одинаковый ответ.
func writeUnauthorized(w http.ResponseWriter, cause string, err error) {
	logCause(cause, err)
	response.WriteError(w, http.StatusUnauthorized,
		models.CodeUnauthorized, models.MsgUnauthorized)
}

// Неизвестный email и неверный пароль дают одинаковый ответ.
func writeInvalidCredentials(w http.ResponseWriter, cause string) {
	logCause(cause, nil)
	response.WriteError(w, http.StatusUnauthorized,
		models.CodeInvalidCredentials, models.MsgInvalidCredentials)
}

func writeAuthError(w http.ResponseWriter, err error) {
	var validationErr *authusecase.ValidationError
	switch {
	case errors.As(err, &validationErr):
		response.WriteFieldsError(w, http.StatusBadRequest, models.CodeValidationFailed, models.MsgValidationFailed, validationErr.Fields)
	case errors.Is(err, models.ErrEmailTaken):
		response.WriteError(w, http.StatusConflict, models.CodeEmailTaken, models.MsgEmailTaken)
	case errors.Is(err, authusecase.ErrInvalidCredentials):
		writeInvalidCredentials(w, err.Error())
	case errors.Is(err, authusecase.ErrUnauthorized):
		writeUnauthorized(w, errSessionRejected, err)
	default:
		writeInternalError(w, "authentication failed", err)
	}
}
