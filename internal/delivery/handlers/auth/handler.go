package auth

import (
	"context"

	"2026_2_Na_Vse_200/internal/config"
	"2026_2_Na_Vse_200/internal/models"
	authusecase "2026_2_Na_Vse_200/internal/usecase/auth"
)

type Usecase interface {
	Signup(context.Context, authusecase.SignupInput) (authusecase.Result, error)
	Login(context.Context, authusecase.LoginInput) (authusecase.Result, error)
	Me(context.Context, models.SessionID) (models.User, error)
	Logout(context.Context, models.SessionID) error
}

type Handler struct {
	cfg     *config.Config
	usecase Usecase
}

func New(cfg *config.Config, usecase Usecase) *Handler {
	return &Handler{cfg: cfg, usecase: usecase}
}
