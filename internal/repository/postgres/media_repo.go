package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository"
)

const createMediaFileQuery = `
	INSERT INTO media_file (storage_key, mime_type, byte_size, duration_ms)
	VALUES ($1, $2, $3, $4)
	RETURNING id::text, created_at, updated_at`

const getMediaFileByIDQuery = `
	SELECT id::text, storage_key, mime_type, byte_size, duration_ms, created_at, updated_at
	FROM media_file
	WHERE id = $1::uuid`

// MediaRepo хранит метаданные файлов; байты обрабатываются файловым слоем.
type MediaRepo struct {
	pool *pgxpool.Pool
}

var _ repository.MediaRepositoryInterface = (*MediaRepo)(nil)

// NewMediaRepo создаёт репозиторий метаданных поверх пула PostgreSQL.
func NewMediaRepo(pool *pgxpool.Pool) *MediaRepo {
	return &MediaRepo{pool: pool}
}

// Create сохраняет метаданные уже сохранённого и проверенного файловым слоем файла.
func (r *MediaRepo) Create(ctx context.Context, file models.MediaFile) (models.MediaFile, error) {
	err := r.pool.QueryRow(ctx, createMediaFileQuery,
		file.StorageKey, file.MIMEType, file.ByteSize, file.DurationMS,
	).Scan(&file.ID, &file.CreatedAt, &file.UpdatedAt)
	if err != nil {
		return models.MediaFile{}, fmt.Errorf("insert media file: %w", err)
	}
	return file, nil
}

// GetByID возвращает метаданные файла или ошибку БД.
func (r *MediaRepo) GetByID(ctx context.Context, id models.MediaFileID) (models.MediaFile, error) {
	var file models.MediaFile
	err := r.pool.QueryRow(ctx, getMediaFileByIDQuery, string(id)).Scan(
		&file.ID,
		&file.StorageKey,
		&file.MIMEType,
		&file.ByteSize,
		&file.DurationMS,
		&file.CreatedAt,
		&file.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidID(err) {
			return models.MediaFile{}, repository.ErrMediaFileNotFound
		}
		return models.MediaFile{}, fmt.Errorf("select media file: %w", err)
	}
	return file, nil
}
