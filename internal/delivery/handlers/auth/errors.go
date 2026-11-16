package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
	authusecase "2026_2_Na_Vse_200/internal/usecase/auth"

	"github.com/rs/zerolog"
)

// Ошибки клиента пишем на INFO, сбои приложения — на ERROR.
func (*Handler) logError(r *http.Request, operation string, err error) {
	var validationErr *authusecase.ValidationError
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	var sizeErr *http.MaxBytesError
	level := zerolog.ErrorLevel
	if errors.As(err, &validationErr) || errors.As(err, &syntaxErr) ||
		errors.As(err, &typeErr) || errors.As(err, &sizeErr) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, models.ErrEmailTaken) || errors.Is(err, authusecase.ErrInvalidCredentials) ||
		errors.Is(err, authusecase.ErrUnauthorized) {
		level = zerolog.InfoLevel
	}
	zerolog.Ctx(r.Context()).WithLevel(level).Str("operation", operation).Err(err).Msg(operation)
}

func writeAuthError(ctx context.Context, w http.ResponseWriter, err error) {
	var validationErr *authusecase.ValidationError
	switch {
	case errors.As(err, &validationErr):
		response.WriteFieldsError(ctx, w, http.StatusBadRequest, models.CodeValidationFailed, models.MsgValidationFailed, validationErr.Fields)
	case errors.Is(err, models.ErrEmailTaken):
		response.WriteError(ctx, w, http.StatusConflict, models.CodeEmailTaken, models.MsgEmailTaken)
	case errors.Is(err, authusecase.ErrInvalidCredentials):
		response.WriteError(ctx, w, http.StatusUnauthorized, models.CodeInvalidCredentials, models.MsgInvalidCredentials)
	case errors.Is(err, authusecase.ErrUnauthorized):
		response.WriteError(ctx, w, http.StatusUnauthorized, models.CodeUnauthorized, models.MsgUnauthorized)
	default:
		response.WriteError(ctx, w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
	}
}
