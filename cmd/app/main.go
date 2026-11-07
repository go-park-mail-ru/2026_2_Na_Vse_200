package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"

	"2026_2_Na_Vse_200/internal/app"
	"2026_2_Na_Vse_200/internal/config"
	applog "2026_2_Na_Vse_200/internal/logger"
	"2026_2_Na_Vse_200/internal/migration"
)

const usage = "usage: app server | migrate <up|version|down steps>"

func main() {
	startupLogger := zerolog.New(os.Stderr).With().Timestamp().Str("service", "music-api").Logger()
	if err := execute(os.Args[1:]); err != nil {
		startupLogger.Error().Err(err).Msg("команда завершилась с ошибкой")
		os.Exit(1)
	}
}

func execute(args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return run(ctx, args)
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	switch args[0] {
	case "-h", "--help", "help":
		if len(args) != 1 {
			return errors.New(usage)
		}
		fmt.Println(usage)
		return nil
	case "server":
		if len(args) != 1 {
			return errors.New("server does not accept arguments")
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("конфигурация: %w", err)
		}

		logger, err := applog.New(
			os.Stdout,
			cfg.LogLevel,
			cfg.LogFormat,
		)
		if err != nil {
			return fmt.Errorf("настройка логирования: %w", err)
		}

		return app.Run(ctx, cfg, logger)
	case "migrate":
		return migration.Run(ctx, args[1:])
	default:
		return fmt.Errorf("unknown command %q; %s", args[0], usage)
	}
}
