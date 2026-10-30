package postgres

import (
	"context"
	"errors"
	"fmt"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MediaRepo struct{ pool *pgxpool.Pool }

var _ ports.MediaRepository = (*MediaRepo)(nil)

func NewMediaRepo(pool *pgxpool.Pool) *MediaRepo { return &MediaRepo{pool: pool} }

func (r *MediaRepo) Create(ctx context.Context, file models.MediaFile) (models.MediaFile, error) {
	const query = `
 INSERT INTO media_file (storage_key, mime_type, byte_size, duration_ms)
 VALUES ($1, $2, $3, $4)
 RETURNING id::text, created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, file.StorageKey, file.MIMEType, file.ByteSize, file.DurationMS).
		Scan(&file.ID, &file.CreatedAt, &file.UpdatedAt)
	if err != nil {
		return models.MediaFile{}, fmt.Errorf("create media file: %w", err)
	}
	return file, nil
}

func (r *MediaRepo) GetByID(ctx context.Context, id models.MediaFileID) (models.MediaFile, error) {
	const query = `
 SELECT id::text, storage_key, mime_type, byte_size, duration_ms, created_at, updated_at
 FROM media_file WHERE id = $1::uuid`
	return scanMediaFile(r.pool.QueryRow(ctx, query, string(id)))
}

func scanMediaFile(row pgx.Row) (models.MediaFile, error) {
	var file models.MediaFile
	err := row.Scan(&file.ID, &file.StorageKey, &file.MIMEType, &file.ByteSize, &file.DurationMS, &file.CreatedAt, &file.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidID(err) {
		return models.MediaFile{}, models.ErrMediaFileNotFound
	}
	if err != nil {
		return models.MediaFile{}, fmt.Errorf("read media file: %w", err)
	}
	return file, nil
}
