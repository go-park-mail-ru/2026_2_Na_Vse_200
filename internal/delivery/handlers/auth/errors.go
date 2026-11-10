package auth

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/logger"
	"2026_2_Na_Vse_200/internal/models"
	authusecase "2026_2_Na_Vse_200/internal/usecase/auth"
)

// Ошибки клиента пишем на INFO, сбои приложения — на ERROR.
func (a *Handler) logError(r *http.Request, operation string, err error) {
	var validationErr *authusecase.ValidationError
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	var sizeErr *http.MaxBytesError
	level := slog.LevelError
	if errors.As(err, &validationErr) || errors.As(err, &syntaxErr) ||
		errors.As(err, &typeErr) || errors.As(err, &sizeErr) || errors.Is(err, io.EOF) ||
		errors.Is(err, models.ErrEmailTaken) || errors.Is(err, authusecase.ErrInvalidCredentials) ||
		errors.Is(err, authusecase.ErrUnauthorized) {
		level = slog.LevelInfo
	}
	logger.FromContext(r.Context(), a.log).Log(r.Context(), level, operation, slog.Any("error", err))
}

func writeAuthError(w http.ResponseWriter, err error) {
	var validationErr *authusecase.ValidationError
	switch {
	case errors.As(err, &validationErr):
		response.WriteFieldsError(w, http.StatusBadRequest, models.CodeValidationFailed, models.MsgValidationFailed, validationErr.Fields)
	case errors.Is(err, models.ErrEmailTaken):
		response.WriteError(w, http.StatusConflict, models.CodeEmailTaken, models.MsgEmailTaken)
	case errors.Is(err, authusecase.ErrInvalidCredentials):
		response.WriteError(w, http.StatusUnauthorized, models.CodeInvalidCredentials, models.MsgInvalidCredentials)
	case errors.Is(err, authusecase.ErrUnauthorized):
		response.WriteError(w, http.StatusUnauthorized, models.CodeUnauthorized, models.MsgUnauthorized)
	default:
		response.WriteError(w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
	}
}
