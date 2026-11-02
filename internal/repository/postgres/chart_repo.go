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

const createChartQuery = `
	INSERT INTO chart (name, period_start, period_end, cover_file_id)
	VALUES ($1, $2, $3, $4)
	RETURNING id::text, created_at, updated_at`

const populateChartFromHistoryQuery = `
	INSERT INTO chart_item (chart_id, track_id, position, streams_count)
	SELECT $1::uuid, ranked.track_id, ranked.position::integer, ranked.streams_count
	FROM (
		SELECT track_id, count(*)::bigint AS streams_count,
			row_number() OVER (ORDER BY count(*) DESC, track_id) AS position
		FROM listening_event
		WHERE started_at >= $2 AND started_at < $3
		GROUP BY track_id
		ORDER BY count(*) DESC, track_id
		LIMIT $4
	) AS ranked`

const getChartQuery = `
	SELECT c.id::text, c.name, c.period_start, c.period_end, c.cover_file_id::text,
		c.created_at, c.updated_at, ci.track_id::text, ci.position, ci.streams_count
	FROM chart AS c
	LEFT JOIN chart_item AS ci ON ci.chart_id = c.id
	WHERE c.id = $1::uuid
	ORDER BY ci.position`

const listChartsQuery = `
	SELECT id::text, name, period_start, period_end, cover_file_id::text, created_at, updated_at
	FROM chart
	ORDER BY period_end DESC, id
	LIMIT $1 OFFSET $2`

const getChartItemsQuery = `
	SELECT track_id::text, position, streams_count
	FROM chart_item
	WHERE chart_id = $1::uuid
	ORDER BY position`

// ChartRepo snapshots aggregate listening counts into chart_item rows.
type ChartRepo struct {
	pool *pgxpool.Pool
}

var _ repository.ChartRepositoryInterface = (*ChartRepo)(nil)

// NewChartRepo creates a chart repository over the shared pool.
func NewChartRepo(pool *pgxpool.Pool) *ChartRepo {
	return &ChartRepo{pool: pool}
}

// CreateFromHistory saves a chart and its top tracks in one transaction.
// A listening event counts once; the interval includes PeriodStart and excludes PeriodEnd.
func (r *ChartRepo) CreateFromHistory(
	ctx context.Context,
	chart models.Chart,
	limit int,
) (result models.Chart, resultErr error) {
	if limit <= 0 {
		return models.Chart{}, errors.New("chart limit must be positive")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.Chart{}, fmt.Errorf("begin create chart: %w", err)
	}
	defer rollback(ctx, tx, &resultErr)

	if err := tx.QueryRow(ctx, createChartQuery,
		chart.Name, chart.PeriodStart, chart.PeriodEnd, mediaFileIDValue(chart.CoverFileID),
	).Scan(&chart.ID, &chart.CreatedAt, &chart.UpdatedAt); err != nil {
		return models.Chart{}, fmt.Errorf("insert chart: %w", err)
	}
	if _, err := tx.Exec(ctx, populateChartFromHistoryQuery,
		string(chart.ID), chart.PeriodStart, chart.PeriodEnd, limit,
	); err != nil {
		return models.Chart{}, fmt.Errorf("populate chart from listening history: %w", err)
	}
	chart.Items, err = readChartItems(ctx, tx, chart.ID)
	if err != nil {
		return models.Chart{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.Chart{}, fmt.Errorf("commit create chart: %w", err)
	}
	return chart, nil
}

func (r *ChartRepo) GetByID(ctx context.Context, id models.ChartID) (models.Chart, error) {
	rows, err := r.pool.Query(ctx, getChartQuery, string(id))
	if err != nil {
		if isInvalidID(err) {
			return models.Chart{}, repository.ErrChartNotFound
		}
		return models.Chart{}, fmt.Errorf("select chart: %w", err)
	}
	defer rows.Close()

	chart := models.Chart{Items: make([]models.ChartItem, 0)}
	found := false
	for rows.Next() {
		var coverID, trackID *string
		var position *int32
		var streamsCount *int64
		if err := rows.Scan(
			&chart.ID, &chart.Name, &chart.PeriodStart, &chart.PeriodEnd, &coverID,
			&chart.CreatedAt, &chart.UpdatedAt, &trackID, &position, &streamsCount,
		); err != nil {
			return models.Chart{}, fmt.Errorf("scan chart: %w", err)
		}
		found = true
		if coverID != nil {
			fileID := models.MediaFileID(*coverID)
			chart.CoverFileID = &fileID
		}
		if trackID != nil && position != nil && streamsCount != nil {
			chart.Items = append(chart.Items, models.ChartItem{
				TrackID: models.TrackID(*trackID), Position: int(*position), StreamsCount: *streamsCount,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return models.Chart{}, fmt.Errorf("read chart: %w", err)
	}
	if !found {
		return models.Chart{}, repository.ErrChartNotFound
	}
	return chart, nil
}

func (r *ChartRepo) List(ctx context.Context, limit, offset int) ([]models.Chart, error) {
	charts := make([]models.Chart, 0)
	if limit <= 0 {
		return charts, nil
	}
	if offset < 0 {
		return nil, errors.New("chart offset cannot be negative")
	}

	rows, err := r.pool.Query(ctx, listChartsQuery, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("select charts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var chart models.Chart
		var coverID *string
		if err := rows.Scan(
			&chart.ID, &chart.Name, &chart.PeriodStart, &chart.PeriodEnd,
			&coverID, &chart.CreatedAt, &chart.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan chart summary: %w", err)
		}
		if coverID != nil {
			fileID := models.MediaFileID(*coverID)
			chart.CoverFileID = &fileID
		}
		charts = append(charts, chart)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read charts: %w", err)
	}
	return charts, nil
}

func readChartItems(ctx context.Context, tx pgx.Tx, chartID models.ChartID) ([]models.ChartItem, error) {
	rows, err := tx.Query(ctx, getChartItemsQuery, string(chartID))
	if err != nil {
		return nil, fmt.Errorf("select chart items: %w", err)
	}
	defer rows.Close()

	items := make([]models.ChartItem, 0)
	for rows.Next() {
		var item models.ChartItem
		if err := rows.Scan(&item.TrackID, &item.Position, &item.StreamsCount); err != nil {
			return nil, fmt.Errorf("scan chart item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read chart items: %w", err)
	}
	return items, nil
}
