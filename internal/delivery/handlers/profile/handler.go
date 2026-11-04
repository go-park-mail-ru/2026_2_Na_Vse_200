package profile

import (
	"context"
	"errors"
	"log"
	"net/http"
	"uuid"

	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
	profileusecase "2026_2_Na_Vse_200/internal/usecase/profile"
)

type Usecase interface {
	Get(ctx context.Context, sessionUserID, requestedID models.UserID) (models.User, error)
}

type Handler struct {
	usecase Usecase
}

func New(usecase Usecase) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.SessionFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, models.CodeUnauthorized, models.MsgUnauthorized)
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.WriteError(w, http.StatusNotFound, models.CodeNotFound, models.MsgNotFound)
		return
	}

	user, err := h.usecase.Get(r.Context(), session.UserID, models.UserID(id.String()))
	switch {
	case errors.Is(err, profileusecase.ErrForbidden):
		response.WriteError(w, http.StatusForbidden, models.CodeForbidden, models.MsgForbidden)
		return
	case errors.Is(err, profileusecase.ErrUnauthorized):
		response.WriteError(w, http.StatusUnauthorized, models.CodeUnauthorized, models.MsgUnauthorized)
		return
	case err != nil:
		log.Printf("чтение профиля: %v", err)
		response.WriteError(w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
		return
	}

	response.WriteJSON(w, http.StatusOK, newProfileResponse(user))
}
