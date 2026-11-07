package profile

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"2026_2_Na_Vse_200/internal/models"
)

type userRepoStub struct {
	user   models.User
	err    error
	calls  int
	failOn bool
}

func (s *userRepoStub) Create(context.Context, models.User) (models.User, error) {
	return models.User{}, nil
}
func (s *userRepoStub) GetByEmail(context.Context, string) (models.User, error) {
	return models.User{}, nil
}
func (s *userRepoStub) GetByID(_ context.Context, id models.UserID) (models.User, error) {
	s.calls++
	if s.failOn {
		return models.User{}, s.err
	}
	user := s.user
	user.ID = id
	return user, s.err
}

type profileRepoStub struct {
	user userRepoStub
}

func (s *profileRepoStub) UpdateProfile(_ context.Context, id models.UserID, email, displayName string) (models.User, error) {
	if s.user.failOn {
		return models.User{}, s.user.err
	}
	user := s.user.user
	user.ID = id
	user.Email = email
	user.DisplayName = displayName
	return user, nil
}

func (s *profileRepoStub) SetAvatar(_ context.Context, id models.UserID, fileID models.MediaFileID) (models.User, error) {
	if s.user.failOn {
		return models.User{}, s.user.err
	}
	user := s.user.user
	user.ID = id
	user.AvatarFileID = &fileID
	return user, nil
}

type mediaRepoStub struct {
	file models.MediaFile
	err  error
}

func (s *mediaRepoStub) Create(_ context.Context, file models.MediaFile) (models.MediaFile, error) {
	if s.err != nil {
		return models.MediaFile{}, s.err
	}
	file.ID = "33333333-3333-3333-3333-333333333333"
	s.file = file
	return file, nil
}

func (s *mediaRepoStub) GetByID(context.Context, models.MediaFileID) (models.MediaFile, error) {
	return s.file, nil
}

type blobStub struct {
	saved   map[string]string
	removed []string
	err     error
}

func (s *blobStub) Save(key string, r io.Reader) error {
	if s.err != nil {
		return s.err
	}
	if s.saved == nil {
		s.saved = map[string]string{}
	}
	data, _ := io.ReadAll(r)
	s.saved[key] = string(data)
	return nil
}

func (s *blobStub) Remove(key string) error {
	s.removed = append(s.removed, key)
	return nil
}

func TestGet(t *testing.T) {
	owner := models.UserID("11111111-1111-1111-1111-111111111111")
	other := models.UserID("22222222-2222-2222-2222-222222222222")
	avatar := models.MediaFileID("44444444-4444-4444-4444-444444444444")
	users := &userRepoStub{user: models.User{Email: "andrey@example.com", DisplayName: "Андрей", AvatarFileID: &avatar}}
	uc := NewProfileUsecase(users, &profileRepoStub{user: *users}, &mediaRepoStub{}, &blobStub{}, nil)

	t.Run("чужой id", func(t *testing.T) {
		_, err := uc.Get(context.Background(), owner, other)
		if !errors.Is(err, models.ErrForbidden) || users.calls != 0 {
			t.Fatalf("err=%v calls=%d", err, users.calls)
		}
	})

	t.Run("владелец", func(t *testing.T) {
		got, err := uc.Get(context.Background(), owner, owner)
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		if got.AvatarURL != avatarURLPath+string(avatar) {
			t.Fatalf("avatar_url=%q", got.AvatarURL)
		}
	})
}

func TestUpdate(t *testing.T) {
	owner := models.UserID("11111111-1111-1111-1111-111111111111")
	users := &userRepoStub{user: models.User{Email: "old@example.com", DisplayName: "Старое"}}
	uc := NewProfileUsecase(users, &profileRepoStub{user: *users}, &mediaRepoStub{}, &blobStub{}, nil)

	got, err := uc.Update(context.Background(), owner, owner, ProfileUpdateInput{
		Email: "Old@Example.com", DisplayName: "Андрей",
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got.Email != "old@example.com" || got.DisplayName != "Андрей" {
		t.Fatalf("got=%+v", got)
	}
}

func TestUploadAvatar(t *testing.T) {
	owner := models.UserID("11111111-1111-1111-1111-111111111111")
	users := &userRepoStub{user: models.User{Email: "a@b.co", DisplayName: "A"}}
	media := &mediaRepoStub{}
	blobs := &blobStub{}
	uc := NewProfileUsecase(users, &profileRepoStub{user: *users}, media, blobs, nil)

	pngHeader := append([]byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
	}, bytes.Repeat([]byte{0}, 16)...)
	got, err := uc.UploadAvatar(context.Background(), owner, owner, "image/png", int64(len(pngHeader)), bytes.NewReader(pngHeader))
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got.AvatarFileID == nil || got.AvatarURL == "" {
		t.Fatalf("avatar not set: %+v", got)
	}
	if len(blobs.saved) != 1 {
		t.Fatalf("saved=%v", blobs.saved)
	}
	if !strings.HasPrefix(media.file.StorageKey, "images/avatars/") {
		t.Fatalf("key=%q", media.file.StorageKey)
	}
}

func TestDetectAvatarMIME(t *testing.T) {
	png := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	mime, err := DetectAvatarMIME(png)
	if err != nil || mime != "image/png" {
		t.Fatalf("mime=%q err=%v", mime, err)
	}
	if _, err := DetectAvatarMIME([]byte("not-an-image")); !errors.Is(err, models.ErrInvalidAvatar) {
		t.Fatalf("err=%v", err)
	}
}
