package auth

import (
	"context"
	"errors"
	"net/http"

	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
	authusecase "2026_2_Na_Vse_200/internal/usecase/auth"
	"github.com/rs/zerolog"
)

const (
	errNoSessionCookie = "запрос без cookie сессии"
	errSessionRejected = "сессия неизвестна или истекла"
)

func writeInternalError(ctx context.Context, w http.ResponseWriter, cause string, err error) {
	zerolog.Ctx(ctx).Error().Err(err).Str("operation", "auth").Msg(cause)
	response.WriteError(ctx, w, http.StatusInternalServerError,
		models.CodeInternal, models.MsgInternal)
}

// Отсутствующая, неизвестная и истёкшая сессии дают одинаковый ответ.
func writeUnauthorized(ctx context.Context, w http.ResponseWriter, cause string, err error) {
	zerolog.Ctx(ctx).Debug().Err(err).Str("operation", "auth").Msg(cause)
	response.WriteError(ctx, w, http.StatusUnauthorized,
		models.CodeUnauthorized, models.MsgUnauthorized)
}

// Неизвестный email и неверный пароль дают одинаковый ответ.
func writeInvalidCredentials(ctx context.Context, w http.ResponseWriter, cause string) {
	zerolog.Ctx(ctx).Debug().Str("operation", "auth").Msg(cause)
	response.WriteError(ctx, w, http.StatusUnauthorized,
		models.CodeInvalidCredentials, models.MsgInvalidCredentials)
}

func writeAuthError(ctx context.Context, w http.ResponseWriter, err error) {
	var validationErr *authusecase.ValidationError
	switch {
	case errors.As(err, &validationErr):
		response.WriteFieldsError(ctx, w, http.StatusBadRequest, models.CodeValidationFailed, models.MsgValidationFailed, validationErr.Fields)
	case errors.Is(err, models.ErrEmailTaken):
		response.WriteError(ctx, w, http.StatusConflict, models.CodeEmailTaken, models.MsgEmailTaken)
	case errors.Is(err, authusecase.ErrInvalidCredentials):
		writeInvalidCredentials(ctx, w, err.Error())
	case errors.Is(err, authusecase.ErrUnauthorized):
		writeUnauthorized(ctx, w, errSessionRejected, err)
	default:
		writeInternalError(ctx, w, "authentication failed", err)
	}
}
