package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
)

type usersStub struct {
	ports.UserRepository
	create  func(context.Context, models.User) (models.User, error)
	byEmail func(context.Context, string) (models.User, error)
	byID    func(context.Context, models.UserID) (models.User, error)
}

func (s usersStub) Create(ctx context.Context, v models.User) (models.User, error) {
	return s.create(ctx, v)
}
func (s usersStub) GetByEmail(ctx context.Context, v string) (models.User, error) {
	return s.byEmail(ctx, v)
}
func (s usersStub) GetByID(ctx context.Context, v models.UserID) (models.User, error) {
	return s.byID(ctx, v)
}

type sessionsStub struct {
	ports.SessionRepository
	create func(context.Context, models.Session) error
	get    func(context.Context, models.SessionID) (models.Session, error)
	delete func(context.Context, models.SessionID) error
}

func (s sessionsStub) Create(ctx context.Context, v models.Session) error { return s.create(ctx, v) }
func (s sessionsStub) GetByID(ctx context.Context, v models.SessionID) (models.Session, error) {
	return s.get(ctx, v)
}
func (s sessionsStub) Delete(ctx context.Context, v models.SessionID) error { return s.delete(ctx, v) }

type hasherStub struct {
	hash   func(string) (string, error)
	verify func(string, string) (bool, error)
}

func (h hasherStub) Hash(v string) (string, error)       { return h.hash(v) }
func (h hasherStub) Verify(v, hash string) (bool, error) { return h.verify(v, hash) }

func newUsecase(users ports.UserRepository, sessions ports.SessionRepository, hasher ports.PasswordHasher) *Usecase {
	return New(users, sessions, hasher, func() models.SessionID { return "session" }, func() time.Time { return time.Unix(1000, 0) }, time.Hour)
}

func TestSignupNormalizesAndCreatesSession(t *testing.T) {
	ctx := context.WithValue(context.Background(), struct{}{}, "request")
	users := usersStub{create: func(gotCtx context.Context, v models.User) (models.User, error) {
		if gotCtx != ctx || v.Email != "user@example.com" || v.DisplayName != "User" || v.PasswordHash != "hash" {
			t.Fatalf("unexpected create: %+v", v)
		}
		v.ID = "user"
		return v, nil
	}}
	var saved models.Session
	sessions := sessionsStub{create: func(gotCtx context.Context, v models.Session) error {
		if gotCtx != ctx {
			t.Fatal("context lost")
		}
		saved = v
		return nil
	}}
	hasher := hasherStub{hash: func(v string) (string, error) {
		if v != "password1" {
			t.Fatalf("password: %q", v)
		}
		return "hash", nil
	}}
	result, err := newUsecase(users, sessions, hasher).Signup(ctx, SignupInput{Email: " USER@example.com ", DisplayName: " User ", Password: " password1 "})
	if err != nil {
		t.Fatal(err)
	}
	if result.User.ID != "user" || saved.UserID != "user" || saved.ID != "session" || !saved.ExpiresAt.Equal(time.Unix(1000, 0).Add(time.Hour)) || result.Session != saved {
		t.Fatalf("result: %+v; session: %+v", result, saved)
	}
}

func TestValidationBeforeDependencies(t *testing.T) {
	u := newUsecase(nil, nil, nil)
	_, err := u.Signup(context.Background(), SignupInput{})
	var fields *ValidationError
	if !errors.As(err, &fields) || len(fields.Fields) != 3 {
		t.Fatalf("signup: %v", err)
	}
	_, err = u.Login(context.Background(), LoginInput{})
	if !errors.As(err, &fields) || len(fields.Fields) != 2 {
		t.Fatalf("login: %v", err)
	}
}

func TestSignupFailures(t *testing.T) {
	failure := errors.New("storage or hash failure")
	for _, stage := range []string{"hash", "user", "session"} {
		t.Run(stage, func(t *testing.T) {
			hasher := hasherStub{hash: func(string) (string, error) {
				if stage == "hash" {
					return "", failure
				}
				return "hash", nil
			}}
			users := usersStub{create: func(context.Context, models.User) (models.User, error) {
				if stage == "user" {
					return models.User{}, models.ErrEmailTaken
				}
				return models.User{ID: "user"}, nil
			}}
			sessions := sessionsStub{create: func(context.Context, models.Session) error { return failure }}
			_, err := newUsecase(users, sessions, hasher).Signup(context.Background(), SignupInput{Email: "user@example.com", DisplayName: "User", Password: "password1"})
			want := failure
			if stage == "user" {
				want = models.ErrEmailTaken
			}
			if !errors.Is(err, want) {
				t.Fatalf("got %v, want %v", err, want)
			}
		})
	}
}

func TestLoginOutcomes(t *testing.T) {
	failure := errors.New("adapter down")
	for _, tc := range []struct {
		name                               string
		userErr, errorVerify, errorSession error
		matched                            bool
		want                               error
	}{
		{name: "unknown email", userErr: models.ErrUserNotFound, want: ErrInvalidCredentials},
		{name: "wrong password", want: ErrInvalidCredentials},
		{name: "user storage", userErr: failure, want: failure},
		{name: "hasher", errorVerify: failure, want: failure},
		{name: "session storage", matched: true, errorSession: failure, want: failure},
		{name: "success", matched: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			users := usersStub{byEmail: func(_ context.Context, email string) (models.User, error) {
				if email != "user@example.com" {
					t.Fatal(email)
				}
				return models.User{ID: "user", PasswordHash: "hash"}, tc.userErr
			}}
			hasher := hasherStub{verify: func(password, hash string) (bool, error) {
				if password != "pass" || hash != "hash" {
					t.Fatal("incorrect hasher input")
				}
				return tc.matched, tc.errorVerify
			}}
			sessions := sessionsStub{create: func(_ context.Context, s models.Session) error {
				if s.UserID != "user" {
					t.Fatal(s)
				}
				return tc.errorSession
			}}
			result, err := newUsecase(users, sessions, hasher).Login(context.Background(), LoginInput{Email: " USER@example.com ", Password: " pass "})
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			if err == nil && result.Session.ID != "session" {
				t.Fatal(result)
			}
		})
	}
}

func TestMeOutcomes(t *testing.T) {
	failure := errors.New("database down")
	for _, tc := range []struct {
		name                string
		sessionErr, userErr error
		expires             time.Time
		want                error
	}{
		{"unknown", models.ErrSessionNotFound, nil, time.Unix(2000, 0), ErrUnauthorized},
		{"expired repository", models.ErrSessionExpired, nil, time.Unix(2000, 0), ErrUnauthorized},
		{"expired domain", nil, nil, time.Unix(1000, 0), ErrUnauthorized},
		{"orphan", nil, models.ErrUserNotFound, time.Unix(2000, 0), ErrUnauthorized},
		{"session storage", failure, nil, time.Unix(2000, 0), failure},
		{"user storage", nil, failure, time.Unix(2000, 0), failure},
		{"success", nil, nil, time.Unix(2000, 0), nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sessions := sessionsStub{get: func(context.Context, models.SessionID) (models.Session, error) {
				return models.Session{UserID: "user", ExpiresAt: tc.expires}, tc.sessionErr
			}}
			users := usersStub{byID: func(_ context.Context, id models.UserID) (models.User, error) {
				if id != "user" {
					t.Fatal(id)
				}
				return models.User{ID: id}, tc.userErr
			}}
			result, err := newUsecase(users, sessions, nil).Me(context.Background(), "session")
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			if err == nil && result.ID != "user" {
				t.Fatal(result)
			}
		})
	}
	if _, err := newUsecase(nil, nil, nil).Me(context.Background(), ""); !errors.Is(err, ErrUnauthorized) {
		t.Fatal(err)
	}
}

func TestLogout(t *testing.T) {
	if err := newUsecase(nil, nil, nil).Logout(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	for _, want := range []error{nil, errors.New("storage down")} {
		sessions := sessionsStub{delete: func(_ context.Context, id models.SessionID) error {
			if id != "session" {
				t.Fatal(id)
			}
			return want
		}}
		if err := newUsecase(nil, sessions, nil).Logout(context.Background(), "session"); !errors.Is(err, want) {
			t.Fatalf("got %v want %v", err, want)
		}
	}
}
