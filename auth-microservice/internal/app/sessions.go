package app

import (
	"context"
	"time"

	"2026_2_Na_Vse_200/auth-microservice/repository"

	"github.com/rs/zerolog"
)

// cleanupInactiveSessions удаляет истёкшие и отозванные сессии при запуске и раз в час.
func cleanupInactiveSessions(ctx context.Context, logger zerolog.Logger, sessions *repository.SessionRepo) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		cleanupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := sessions.DeleteInactive(cleanupCtx)
		cancel()
		if err != nil && ctx.Err() == nil {
			logger.Error().Err(err).Str("operation", "sessions.cleanup").Msg("не удалось очистить неактивные сессии")
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
