package postgres

import (
	"2026_2_Na_Vse_200/internal/models"
	"context"
)

// FavoriteIDs returns only the authenticated account's favorites.
func (r *TrackRepo) FavoriteIDs(ctx context.Context, account models.UserID) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT f.track_id::text FROM favorite_track f JOIN track t ON t.id=f.track_id WHERE f.account_id=$1::uuid AND t.status='published' ORDER BY f.track_id`, string(account))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
func (r *TrackRepo) SetFavorite(ctx context.Context, account models.UserID, track models.TrackID, liked bool) error {
	if !liked {
		_, err := r.pool.Exec(ctx, `DELETE FROM favorite_track WHERE account_id=$1::uuid AND track_id=$2::uuid`, string(account), string(track))
		return err
	}
	var exists bool
	err := r.pool.QueryRow(ctx, `WITH published AS (SELECT id FROM track WHERE id=$2::uuid AND status='published'), inserted AS (INSERT INTO favorite_track(account_id,track_id) SELECT $1::uuid,id FROM published ON CONFLICT DO NOTHING RETURNING track_id) SELECT EXISTS(SELECT 1 FROM published)`, string(account), string(track)).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return models.ErrTrackNotFound
	}
	return nil
}
