package profile

import (
	"fmt"
	"time"
	"uuid"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/security"
)

type profileResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url"`
	CreatedAt   time.Time `json:"created_at"`
}

func newProfileResponse(user models.User) (profileResponse, error) {
	id, err := idToString(string(user.ID))
	if err != nil {
		return profileResponse{}, fmt.Errorf("user id: %w", err)
	}
	resp := profileResponse{
		ID:          id,
		Email:       user.Email,
		DisplayName: security.EscapeText(user.DisplayName),
		CreatedAt:   user.CreatedAt.UTC(),
	}
	if user.AvatarURL != "" {
		avatar := user.AvatarURL
		resp.AvatarURL = &avatar
	}
	return resp, nil
}

func idToString(raw string) (string, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return "", err
	}
	return id.String(), nil
}
