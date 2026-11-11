package auth

import (
	"context"
	"log/slog"

	"2026_2_Na_Vse_200/internal/config"
	"2026_2_Na_Vse_200/internal/models"
	authusecase "2026_2_Na_Vse_200/internal/usecase/auth"
)

type Usecase interface {
	Signup(context.Context, authusecase.SignupInput) (authusecase.Result, error)
	Login(context.Context, authusecase.LoginInput) (authusecase.Result, error)
	Me(context.Context, models.SessionID) (models.User, error)
	Logout(context.Context, models.SessionID) error
	SessionByID(context.Context, models.SessionID) (models.Session, error)
}

type Handler struct {
	cfg     *config.Config
	usecase Usecase
	log     *slog.Logger
}

func New(cfg *config.Config, usecase Usecase, logs ...*slog.Logger) *Handler {
	log := slog.Default()
	if len(logs) > 0 && logs[0] != nil {
		log = logs[0]
	}
	return &Handler{cfg: cfg, usecase: usecase, log: log}
}
