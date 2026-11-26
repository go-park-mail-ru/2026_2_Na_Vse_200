package rpcmetrics

import (
	"context"
	"fmt"
	"time"

	"2026_2_Na_Vse_200/internal/httpserver"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

func (m *Metrics) RunWithServer(
	ctx context.Context,
	addr string,
	timeout time.Duration,
	logger zerolog.Logger,
	run func(context.Context) error,
) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		server := httpserver.New(addr, m.Handler())
		server.WriteTimeout = 5 * time.Second
		if err := httpserver.Serve(ctx, server, timeout, logger); err != nil {
			return fmt.Errorf("сервер метрик: %w", err)
		}
		return nil
	})
	group.Go(func() error {
		defer cancel()
		return run(ctx)
	})
	return group.Wait()
}
