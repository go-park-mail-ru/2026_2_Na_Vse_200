package app

import (
	"context"
	"log/slog"
	"time"

	"2026_2_Na_Vse_200/internal/repository/postgres"
)

// cleanupExpiredSessions удаляет истёкшие сессии при запуске и раз в час.
func cleanupExpiredSessions(ctx context.Context, logger *slog.Logger, sessions *postgres.SessionRepo) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		cleanupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := sessions.DeleteExpired(cleanupCtx)
		cancel()
		if err != nil && ctx.Err() == nil {
			logger.Error("очистка истёкших сессий", slog.Any("error", err))
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
