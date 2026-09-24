// Команда server поднимает HTTP API музыкального сервиса.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/config"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/handlers"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("конфигурация: %v", err)
	}

	// Хранилища появятся в BE-03 (аккаунты) и BE-04 (сессии): /health их не использует.
	api := handlers.New(cfg, handlers.Deps{})

	// Порядок обёрток: recover снаружи всех, дальше лог, CORS и подмена
	// текстовых 404/405 на JSON, внутри — сама таблица маршрутов.
	handler := handlers.Wrap(
		api.Routes(),
		handlers.WithRecover,
		handlers.WithLogging,
		handlers.WithCORS(cfg.AllowedOrigins),
		handlers.WithJSONErrors,
	)

	// Свой http.Server, а не http.ListenAndServe(addr, nil): нужен контроль
	// над таймаутами и своя таблица маршрутов вместо глобального DefaultServeMux.
	// Без таймаутов медленный клиент может держать соединение сколько угодно.
	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// NotifyContext отменяет контекст по Ctrl+C или SIGTERM от системы
	// (его шлёт стенд при передеплое).
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ListenAndServe блокирует, поэтому уводим его в отдельную горутину,
	// а в главной ждём сигнала остановки.
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("сервер слушает %s", cfg.Addr)
		// После Shutdown ListenAndServe возвращает ErrServerClosed — это штатное
		// завершение, а не сбой.
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
			return
		}
		serverErrors <- nil
	}()

	select {
	case err := <-serverErrors:
		if err != nil {
			log.Fatalf("сервер остановлен с ошибкой: %v", err)
		}
	case <-ctx.Done():
		log.Println("получен сигнал остановки, завершаем запросы")

		// Shutdown перестаёт принимать новые соединения и ждёт текущие запросы,
		// но не дольше отведённого времени.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("остановка сервера: %v", err)
		}
		log.Println("сервер остановлен")
	}
}
