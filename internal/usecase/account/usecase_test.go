package account

import (
	"context"
	"errors"
	"testing"

	"2026_2_Na_Vse_200/internal/auth"
	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
)

type usersStub struct {
	ports.UserRepository
	user models.User
	err  error
}

func (s *usersStub) GetByID(context.Context, models.UserID) (models.User, error) {
	return s.user, s.err
}

type accountStub struct {
	calls       int
	email, hash string
}

func (s *accountStub) ChangeEmail(_ context.Context, id models.UserID, hash, email string) (models.User, error) {
	s.calls++
	s.email = email
	return models.User{ID: id, Email: email}, nil
}
func (s *accountStub) ChangePassword(_ context.Context, id models.UserID, previous, next string) error {
	s.calls++
	s.hash = next
	return nil
}
func (s *accountStub) DeleteAccount(context.Context, models.UserID, string) error {
	s.calls++
	return nil
}

func TestAccountChanges(t *testing.T) {
	hasher := auth.NewBcryptHasherWithCost(auth.MinCost)
	hash, err := hasher.Hash("OldPass123")
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"email", "password", "delete"} {
		for _, scenario := range []string{"success", "wrong password", "other user", "missing password", "missing account", "invalid input"} {
			t.Run(action+"/"+scenario, func(t *testing.T) {
				users := &usersStub{user: models.User{ID: "owner", Email: "old@example.test", DisplayName: "Name", PasswordHash: hash}}
				repo := &accountStub{}
				u := New(users, repo, hasher)
				owner, password, value := models.UserID("owner"), "OldPass123", "new@example.test"
				if action == "password" {
					value = "NewPass456"
				}
				switch scenario {
				case "wrong password":
					password = "WrongPass123"
				case "other user":
					owner = "someone-else"
				case "missing password":
					password = ""
				case "missing account":
					users.err = models.ErrUserNotFound
				case "invalid input":
					value = "bad"
				}
				var err error
				switch action {
				case "email":
					_, err = u.ChangeEmail(context.Background(), owner, "owner", password, value)
				case "password":
					err = u.ChangePassword(context.Background(), owner, "owner", password, value)
				case "delete":
					err = u.Delete(context.Background(), owner, "owner", password)
				}
				success := scenario == "success" || action == "delete" && scenario == "invalid input"
				if success {
					if err != nil || repo.calls != 1 {
						t.Fatalf("err=%v calls=%d", err, repo.calls)
					}
					if action == "password" {
						valid, e := hasher.Verify(value, repo.hash)
						if e != nil || !valid || repo.hash == value {
							t.Fatal("password was not hashed")
						}
					}
					return
				}
				if err == nil || repo.calls != 0 {
					t.Fatalf("err=%v calls=%d", err, repo.calls)
				}
				if scenario == "wrong password" && !errors.Is(err, models.ErrCurrentPassword) {
					t.Fatal(err)
				}
				if scenario == "other user" && !errors.Is(err, models.ErrForbidden) {
					t.Fatal(err)
				}
				if scenario == "missing account" && !errors.Is(err, models.ErrUnauthorized) {
					t.Fatal(err)
				}
			})
		}
	}
}
