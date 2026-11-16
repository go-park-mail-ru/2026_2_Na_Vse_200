package auth

import (
	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/security"
)

type userResponse struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

func newUserResponse(user models.User) userResponse {
	resp := userResponse{
		ID:          string(user.ID),
		Email:       user.Email,
		DisplayName: security.EscapeText(user.DisplayName),
	}

	if user.AvatarURL != "" {
		avatar := user.AvatarURL
		resp.AvatarURL = &avatar
	}
	return resp
}
