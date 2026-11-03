package auth

import (
	"2026_2_Na_Vse_200/internal/models"
	"testing"
)

func TestUserResponseAvatar(t *testing.T) {
	const url = "https://cdn.example.com/avatars/andrey.jpg"

	withAvatar := newUserResponse(models.User{AvatarURL: url})
	if withAvatar.AvatarURL == nil {
		t.Fatal("аватар потерялся")
	}

	if *withAvatar.AvatarURL != url {
		t.Errorf("аватар = %q, ожидался %q", *withAvatar.AvatarURL, url)
	}

	if withoutAvatar := newUserResponse(models.User{}); withoutAvatar.AvatarURL != nil {
		t.Errorf("аватар = %q, ожидался nil", *withoutAvatar.AvatarURL)
	}
}
