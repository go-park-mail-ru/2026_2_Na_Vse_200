package profile

import (
	"time"

	"2026_2_Na_Vse_200/internal/models"
)

type profileResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url"`
	CreatedAt   time.Time `json:"created_at"`
}

func newProfileResponse(user models.User) profileResponse {
	resp := profileResponse{
		ID:          string(user.ID),
		Email:       user.Email,
		DisplayName: user.DisplayName,
		CreatedAt:   user.CreatedAt.UTC(),
	}
	if user.AvatarURL != "" {
		avatar := user.AvatarURL
		resp.AvatarURL = &avatar
	}
	return resp
}
