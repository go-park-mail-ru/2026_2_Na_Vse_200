package app

import (
	"context"
	"time"

	"2026_2_Na_Vse_200/internal/repository/postgres"

	"github.com/rs/zerolog"
)

// cleanupExpiredSessions удаляет истёкшие сессии при запуске и раз в час.
func cleanupExpiredSessions(ctx context.Context, logger zerolog.Logger, sessions *postgres.SessionRepo) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		cleanupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := sessions.DeleteExpired(cleanupCtx)
		cancel()
		if err != nil && ctx.Err() == nil {
			logger.Error().Err(err).Str("operation", "sessions.cleanup").Msg("не удалось очистить истёкшие сессии")
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
