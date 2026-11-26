package app

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

func startMetrics(addr string, handler http.Handler, logger zerolog.Logger) (func(), error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("запуск сервера метрик: %w", err)
	}
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", handler)
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 3 * time.Second, WriteTimeout: 10 * time.Second}
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error().Err(err).Msg("сервер метрик остановлен с ошибкой")
		}
	}()
	return func() {
		_ = server.Close()
		<-done
	}, nil
}
