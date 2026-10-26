package postgres

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/repository"
)

const addFavoriteQuery = `
	INSERT INTO favorite_track (account_id, track_id)
	VALUES ($1::uuid, $2::uuid)
	ON CONFLICT (account_id, track_id) DO NOTHING`

const removeFavoriteQuery = `DELETE FROM favorite_track WHERE account_id = $1::uuid AND track_id = $2::uuid`

const listFavoritesQuery = `
	SELECT track_id::text
	FROM favorite_track
	WHERE account_id = $1::uuid
	ORDER BY track_id
	LIMIT $2 OFFSET $3`

const recordListeningEventQuery = `
	INSERT INTO listening_event (id, account_id, track_id, started_at)
	VALUES ($1::uuid, $2::uuid, $3::uuid, COALESCE($4::timestamptz, CURRENT_TIMESTAMP))
	RETURNING started_at`

const listListeningHistoryQuery = `
	SELECT id::text, account_id::text, track_id::text, started_at
	FROM listening_event
	WHERE account_id = $1::uuid
	ORDER BY started_at DESC, id DESC
	LIMIT $2 OFFSET $3`

// FavoriteRepo stores an account's favorite track IDs.
type FavoriteRepo struct {
	pool *pgxpool.Pool
}

var _ repository.FavoriteRepositoryInterface = (*FavoriteRepo)(nil)

// NewFavoriteRepo creates a favorite repository over the shared pool.
func NewFavoriteRepo(pool *pgxpool.Pool) *FavoriteRepo {
	return &FavoriteRepo{pool: pool}
}

// Add is idempotent for an already-favorited track.
func (r *FavoriteRepo) Add(ctx context.Context, accountID models.UserID, trackID models.TrackID) error {
	if _, err := r.pool.Exec(ctx, addFavoriteQuery, string(accountID), string(trackID)); err != nil {
		return fmt.Errorf("add favorite track: %w", err)
	}
	return nil
}

// Remove is idempotent when the favorite does not exist.
func (r *FavoriteRepo) Remove(ctx context.Context, accountID models.UserID, trackID models.TrackID) error {
	if _, err := r.pool.Exec(ctx, removeFavoriteQuery, string(accountID), string(trackID)); err != nil {
		return fmt.Errorf("remove favorite track: %w", err)
	}
	return nil
}

func (r *FavoriteRepo) List(ctx context.Context, accountID models.UserID, limit, offset int) ([]models.TrackID, error) {
	favorites := make([]models.TrackID, 0)
	if limit <= 0 {
		return favorites, nil
	}
	if offset < 0 {
		return nil, errors.New("favorite offset cannot be negative")
	}

	rows, err := r.pool.Query(ctx, listFavoritesQuery, string(accountID), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("select favorite tracks: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan favorite track: %w", err)
		}
		favorites = append(favorites, models.TrackID(id))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read favorite tracks: %w", err)
	}
	return favorites, nil
}

// ListeningRepo writes and reads per-account listening events.
type ListeningRepo struct {
	pool *pgxpool.Pool
}

var _ repository.ListeningHistoryRepositoryInterface = (*ListeningRepo)(nil)

// NewListeningRepo creates a listening-history repository over the shared pool.
func NewListeningRepo(pool *pgxpool.Pool) *ListeningRepo {
	return &ListeningRepo{pool: pool}
}

// Record uses the database clock when StartedAt is zero.
func (r *ListeningRepo) Record(ctx context.Context, event models.ListeningEvent) (models.ListeningEvent, error) {
	if event.ID == "" {
		event.ID = models.ListeningEventID(uuid.New().String())
	}
	var startedAt any
	if !event.StartedAt.IsZero() {
		startedAt = event.StartedAt
	}
	err := r.pool.QueryRow(ctx, recordListeningEventQuery,
		string(event.ID), string(event.AccountID), string(event.TrackID), startedAt,
	).Scan(&event.StartedAt)
	if err != nil {
		return models.ListeningEvent{}, fmt.Errorf("insert listening event: %w", err)
	}
	return event, nil
}

func (r *ListeningRepo) ListByAccount(
	ctx context.Context,
	accountID models.UserID,
	limit, offset int,
) ([]models.ListeningEvent, error) {
	events := make([]models.ListeningEvent, 0)
	if limit <= 0 {
		return events, nil
	}
	if offset < 0 {
		return nil, errors.New("listening history offset cannot be negative")
	}

	rows, err := r.pool.Query(ctx, listListeningHistoryQuery, string(accountID), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("select listening history: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var event models.ListeningEvent
		if err := rows.Scan(&event.ID, &event.AccountID, &event.TrackID, &event.StartedAt); err != nil {
			return nil, fmt.Errorf("scan listening event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read listening history: %w", err)
	}
	return events, nil
}
