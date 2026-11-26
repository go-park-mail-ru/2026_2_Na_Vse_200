package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"2026_2_Na_Vse_200/auth-microservice/internal/app"
	"2026_2_Na_Vse_200/auth-microservice/internal/config"
	"2026_2_Na_Vse_200/pkg/logger"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if len(args) == 2 && args[0] == "-check" {
		return app.CheckAuth(ctx, args[1])
	}
	if len(args) != 0 {
		return fmt.Errorf("usage: auth [-check host:port]")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log, err := logger.NewForService(os.Stdout, cfg.LogLevel, cfg.LogFormat, "music-auth")
	if err != nil {
		return err
	}
	return app.RunAuth(ctx, cfg, log)
}
