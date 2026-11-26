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
	create func(context.Context, models.UserID, int64) (models.Session, error)
	get    func(context.Context, models.SessionID) (models.Session, error)
	delete func(context.Context, models.SessionID) error
}

func (s sessionsStub) CreateSession(ctx context.Context, userID models.UserID, credentialVersion int64) (models.Session, error) {
	return s.create(ctx, userID, credentialVersion)
}

func (s sessionsStub) GetSession(ctx context.Context, id models.SessionID) (models.Session, error) {
	return s.get(ctx, id)
}

func (s sessionsStub) DeleteSession(ctx context.Context, id models.SessionID) error {
	return s.delete(ctx, id)
}

type hasherStub struct {
	hash   func(string) (string, error)
	verify func(string, string) (bool, error)
}

func (h hasherStub) Hash(v string) (string, error)       { return h.hash(v) }
func (h hasherStub) Verify(v, hash string) (bool, error) { return h.verify(v, hash) }

func TestSignupNormalizesAndCreatesSession(t *testing.T) {
	ctx := context.WithValue(context.Background(), struct{}{}, "request")
	users := usersStub{create: func(gotCtx context.Context, v models.User) (models.User, error) {
		if gotCtx != ctx || v.Email != "user@example.com" || v.DisplayName != "User" || v.PasswordHash != "hash" {
			t.Fatalf("unexpected create: %+v", v)
		}
		v.ID = "user"
		v.CredentialVersion = 1
		return v, nil
	}}
	wantSession := models.Session{ID: "session", UserID: "user", ExpiresAt: time.Unix(4600, 0)}
	sessions := sessionsStub{create: func(gotCtx context.Context, userID models.UserID, version int64) (models.Session, error) {
		if gotCtx != ctx || userID != wantSession.UserID || version != 1 {
			t.Fatalf("unexpected session request: user=%q version=%d", userID, version)
		}
		return wantSession, nil
	}}
	hasher := hasherStub{hash: func(v string) (string, error) {
		if v != "password1" {
			t.Fatalf("password: %q", v)
		}
		return "hash", nil
	}}
	result, err := NewAuthUsecase(users, sessions, hasher).Signup(ctx, SignupInput{Email: " USER@example.com ", DisplayName: " User ", Password: " password1 "})
	if err != nil {
		t.Fatal(err)
	}
	if result.User.ID != "user" || result.Session != wantSession {
		t.Fatalf("result: %+v; want session: %+v", result, wantSession)
	}
}

func TestValidationBeforeDependencies(t *testing.T) {
	u := NewAuthUsecase(nil, nil, nil)
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
			sessions := sessionsStub{create: func(context.Context, models.UserID, int64) (models.Session, error) {
				return models.Session{}, failure
			}}
			_, err := NewAuthUsecase(users, sessions, hasher).Signup(context.Background(), SignupInput{Email: "user@example.com", DisplayName: "User", Password: "password1"})
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
		{name: "password changed", matched: true, errorSession: ErrInvalidCredentials, want: ErrInvalidCredentials},
		{name: "success", matched: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			users := usersStub{byEmail: func(_ context.Context, email string) (models.User, error) {
				if email != "user@example.com" {
					t.Fatal(email)
				}
				return models.User{ID: "user", PasswordHash: "hash", CredentialVersion: 4}, tc.userErr
			}}
			hasher := hasherStub{verify: func(password, hash string) (bool, error) {
				if password != "pass" || hash != "hash" {
					t.Fatal("incorrect hasher input")
				}
				return tc.matched, tc.errorVerify
			}}
			sessions := sessionsStub{create: func(_ context.Context, userID models.UserID, version int64) (models.Session, error) {
				if userID != "user" || version != 4 {
					t.Fatalf("unexpected session request: user=%q version=%d", userID, version)
				}
				return models.Session{ID: "session", UserID: userID}, tc.errorSession
			}}
			result, err := NewAuthUsecase(users, sessions, hasher).Login(context.Background(), LoginInput{Email: " USER@example.com ", Password: " pass "})
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
			result, err := NewAuthUsecase(users, sessions, nil).Me(context.Background(), "session")
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			if err == nil && result.ID != "user" {
				t.Fatal(result)
			}
		})
	}
	missingSession := sessionsStub{get: func(_ context.Context, id models.SessionID) (models.Session, error) {
		if id != "" {
			t.Fatalf("unexpected session ID: %q", id)
		}
		return models.Session{}, models.ErrSessionNotFound
	}}
	if _, err := NewAuthUsecase(nil, missingSession, nil).Me(context.Background(), ""); !errors.Is(err, ErrUnauthorized) {
		t.Fatal(err)
	}
}

func TestLogout(t *testing.T) {
	if err := NewAuthUsecase(nil, nil, nil).Logout(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	for _, want := range []error{nil, errors.New("storage down")} {
		sessions := sessionsStub{delete: func(_ context.Context, id models.SessionID) error {
			if id != "session" {
				t.Fatal(id)
			}
			return want
		}}
		if err := NewAuthUsecase(nil, sessions, nil).Logout(context.Background(), "session"); !errors.Is(err, want) {
			t.Fatalf("got %v want %v", err, want)
		}
	}
}
