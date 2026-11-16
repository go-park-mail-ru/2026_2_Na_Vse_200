package auth

import (
	"2026_2_Na_Vse_200/internal/models"
	"testing"
)

func TestUserResponseEscapesDisplayName(t *testing.T) {
	got := newUserResponse(models.User{DisplayName: `<script>alert(1)</script>`})
	if got.DisplayName != `&lt;script&gt;alert(1)&lt;/script&gt;` {
		t.Fatalf("display_name = %q, ожидалось экранирование", got.DisplayName)
	}
}

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
