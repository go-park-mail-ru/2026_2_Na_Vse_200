package profile

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"uuid"

	"2026_2_Na_Vse_200/internal/logger"
	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
	"2026_2_Na_Vse_200/internal/validation"
)

const (
	MaxAvatarBytes = 2_097_152
	avatarURLPath  = "/api/v1/media/"
)

type ProfileUpdateInput struct {
	Email       string
	DisplayName string
}

type ProfileBlobStore interface {
	Save(key string, r io.Reader) error
	Remove(key string) error
}

type ProfileUsecase struct {
	users    ports.UserRepository
	profiles ports.ProfileRepository
	media    ports.MediaRepository
	blobs    ProfileBlobStore
	log      *slog.Logger
}

func NewProfileUsecase(
	users ports.UserRepository,
	profiles ports.ProfileRepository,
	media ports.MediaRepository,
	blobs ProfileBlobStore,
	log *slog.Logger,
) *ProfileUsecase {
	if log == nil {
		log = logger.Discard()
	}
	return &ProfileUsecase{users: users, profiles: profiles, media: media, blobs: blobs, log: log}
}

func (u *ProfileUsecase) Get(ctx context.Context, sessionUserID, requestedID models.UserID) (models.User, error) {
	log := logger.FromContext(ctx, u.log)
	log.Info("profile.Get: старт", slog.String("user_id", string(requestedID)))

	if err := ensureOwner(sessionUserID, requestedID); err != nil {
		log.Info("profile.Get: доступ запрещён", slog.String("session_user_id", string(sessionUserID)))
		return models.User{}, err
	}

	user, err := u.users.GetByID(ctx, sessionUserID)
	if errors.Is(err, models.ErrUserNotFound) {
		log.Info("profile.Get: пользователь не найден", slog.String("user_id", string(sessionUserID)))
		return models.User{}, models.ErrUnauthorized
	}
	if err != nil {
		log.Error("profile.Get: чтение пользователя", slog.Any("error", err), slog.String("user_id", string(sessionUserID)))
		return models.User{}, fmt.Errorf("find user: %w", err)
	}
	log.Info("profile.Get: успех", slog.String("user_id", string(user.ID)))
	return withAvatarURL(user), nil
}

func (u *ProfileUsecase) Update(ctx context.Context, sessionUserID, requestedID models.UserID, in ProfileUpdateInput) (models.User, error) {
	log := logger.FromContext(ctx, u.log)
	log.Info("profile.Update: старт", slog.String("user_id", string(requestedID)))

	if err := ensureOwner(sessionUserID, requestedID); err != nil {
		log.Info("profile.Update: доступ запрещён", slog.String("session_user_id", string(sessionUserID)))
		return models.User{}, err
	}

	form := validation.ProfileUpdate(validation.ProfileUpdateInput(in))
	if len(form.Fields) > 0 {
		log.Info("profile.Update: ошибка валидации", slog.String("user_id", string(sessionUserID)))
		return models.User{}, &models.ValidationError{Fields: form.Fields}
	}

	current, err := u.Get(ctx, sessionUserID, requestedID)
	if err != nil {
		return models.User{}, err
	}
	if form.Email != current.Email {
		return models.User{}, &models.ValidationError{Fields: map[string]string{"email": "Используйте смену email с подтверждением текущего пароля"}}
	}

	user, err := u.profiles.UpdateProfile(ctx, sessionUserID, form.Email, form.DisplayName)
	if errors.Is(err, models.ErrEmailTaken) {
		log.Info("profile.Update: email занят", slog.String("user_id", string(sessionUserID)))
		return models.User{}, models.ErrEmailTaken
	}
	if errors.Is(err, models.ErrUserNotFound) {
		log.Info("profile.Update: пользователь не найден", slog.String("user_id", string(sessionUserID)))
		return models.User{}, models.ErrUnauthorized
	}
	if err != nil {
		log.Error("profile.Update: обновление профиля", slog.Any("error", err), slog.String("user_id", string(sessionUserID)))
		return models.User{}, fmt.Errorf("update profile: %w", err)
	}
	log.Info("profile.Update: успех", slog.String("user_id", string(user.ID)))
	return withAvatarURL(user), nil
}

func (u *ProfileUsecase) UploadAvatar(ctx context.Context, sessionUserID, requestedID models.UserID, mimeType string, size int64, body io.Reader) (models.User, error) {
	log := logger.FromContext(ctx, u.log)
	log.Info("profile.UploadAvatar: старт",
		slog.String("user_id", string(requestedID)),
		slog.String("mime_type", mimeType),
		slog.Int64("size", size),
	)

	if err := ensureOwner(sessionUserID, requestedID); err != nil {
		log.Info("profile.UploadAvatar: доступ запрещён", slog.String("session_user_id", string(sessionUserID)))
		return models.User{}, err
	}
	if size <= 0 || size > MaxAvatarBytes {
		log.Info("profile.UploadAvatar: некорректный размер", slog.Int64("size", size))
		return models.User{}, models.ErrInvalidAvatar
	}

	ext, err := avatarExtension(mimeType)
	if err != nil {
		log.Info("profile.UploadAvatar: некорректный тип файла", slog.String("mime_type", mimeType))
		return models.User{}, err
	}

	key := path.Join("images", "avatars", uuid.New().String()+"."+ext)
	if err := u.blobs.Save(key, io.LimitReader(body, size+1)); err != nil {
		log.Error("profile.UploadAvatar: сохранение файла", slog.Any("error", err), slog.String("key", key))
		return models.User{}, fmt.Errorf("save avatar: %w", err)
	}
	log.Info("profile.UploadAvatar: файл сохранён", slog.String("key", key))

	file, err := u.media.Create(ctx, models.MediaFile{
		StorageKey: key,
		MIMEType:   mimeType,
		ByteSize:   size,
	})
	if err != nil {
		_ = u.blobs.Remove(key)
		log.Error("profile.UploadAvatar: создание media", slog.Any("error", err), slog.String("key", key))
		return models.User{}, fmt.Errorf("create media: %w", err)
	}
	log.Info("profile.UploadAvatar: media создан", slog.String("media_id", string(file.ID)))

	user, err := u.profiles.SetAvatar(ctx, sessionUserID, file.ID)
	if errors.Is(err, models.ErrUserNotFound) {
		_ = u.blobs.Remove(key)
		log.Info("profile.UploadAvatar: пользователь не найден", slog.String("user_id", string(sessionUserID)))
		return models.User{}, models.ErrUnauthorized
	}
	if err != nil {
		_ = u.blobs.Remove(key)
		log.Error("profile.UploadAvatar: установка аватара", slog.Any("error", err), slog.String("user_id", string(sessionUserID)))
		return models.User{}, fmt.Errorf("set avatar: %w", err)
	}
	log.Info("profile.UploadAvatar: успех", slog.String("user_id", string(user.ID)), slog.String("media_id", string(file.ID)))
	return withAvatarURL(user), nil
}

func ensureOwner(sessionUserID, requestedID models.UserID) error {
	if sessionUserID != requestedID {
		return models.ErrForbidden
	}
	return nil
}

func withAvatarURL(user models.User) models.User {
	if user.AvatarFileID != nil {
		user.AvatarURL = avatarURLPath + string(*user.AvatarFileID)
	}
	return user
}

func avatarExtension(mimeType string) (string, error) {
	switch mimeType {
	case "image/jpeg":
		return "jpg", nil
	case "image/png":
		return "png", nil
	case "image/webp":
		return "webp", nil
	default:
		return "", models.ErrInvalidAvatar
	}
}

func DetectAvatarMIME(header []byte) (string, error) {
	mimeType := http.DetectContentType(header)
	if _, err := avatarExtension(mimeType); err != nil {
		return "", models.ErrInvalidAvatar
	}
	return mimeType, nil
}
