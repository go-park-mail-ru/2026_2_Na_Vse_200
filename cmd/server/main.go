// Команда server поднимает HTTP API музыкального сервиса.
package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/auth"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/config"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/handlers"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/storage/memory"
)

// component попадает в каждую строку лога полем handled_by и отвечает
// на вопрос «кто обработал запрос». Пока сервис один; когда появятся
// отдельные сервисы, у каждого будет своё имя.
const component = "monolith/middleware"

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("конфигурация: %v", err)
	}

	// Логи структурированные: не строка текста, а набор полей. Такие записи
	// фильтруются и считаются — например, все запросы со status 500 за час.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Пока аккаунты живут в памяти процесса: это позволяет фронтенду работать
	// до готовности слоя на PostgreSQL. При переходе на него меняется только
	// эта строка — обработчики зависят от интерфейсов, а не от реализации.
	// Плата: после перезапуска сервера аккаунты пропадают.
	users := memory.NewUserStorage()

	api := handlers.New(cfg, handlers.Deps{
		Users:  users,
		Hasher: auth.NewBcryptHasher(),
	})

	// Порядок обёрток: сначала идентификатор запроса (он нужен и логу,
	// и записи о панике), затем recover снаружи остальных, дальше лог,
	// CORS и подмена текстовых 404/405 на JSON, внутри — таблица маршрутов.
	handler := middleware.Chain(
		api.Routes(),
		middleware.WithRequestID,
		middleware.WithRecover(logger),
		middleware.WithLogging(logger, component),
		middleware.WithCORS(cfg.AllowedOrigin),
		middleware.WithJSONErrors,
	)

	// Свой http.Server, а не http.ListenAndServe(addr, nil): нужен контроль
	// над таймаутами и своя таблица маршрутов вместо глобального DefaultServeMux.
	// Без таймаутов медленный клиент может держать соединение сколько угодно,
	// занимая память и файловый дескриптор.
	//
	// Значения подобраны под JSON-API: ответы весят сотни байт, самая долгая
	// операция — проверка пароля, порядка 60 мс. Запас пятикратный с лихвой,
	// а чем быстрее отпускаем зависших клиентов, тем больше живых обслужим.
	// Появится отдача аудиофайлов — для таких ручек таймаут задаётся отдельно.
	server := &http.Server{
		Addr:    cfg.Addr,
		Handler: handler,
		// Заголовки приходят первыми и почти мгновенно: если их нет за три
		// секунды, клиент явно неисправен.
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		// Простаивающее соединение держим дольше: это keep-alive, по нему
		// придёт следующий запрос того же клиента без новых рукопожатий.
		IdleTimeout: 60 * time.Second,
	}

	// NotifyContext отменяет контекст по Ctrl+C или SIGTERM от системы
	// (его шлёт стенд при передеплое).
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ListenAndServe блокирует, поэтому уводим его в отдельную горутину,
	// а в главной ждём сигнала остановки.
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("сервер запущен", slog.String("addr", cfg.Addr), slog.String("component", component))
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
		logger.Info("получен сигнал остановки, завершаем текущие запросы")

		// Shutdown перестаёт принимать новые соединения и ждёт текущие запросы,
		// но не дольше отведённого времени.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("остановка сервера: %v", err)
		}
		logger.Info("сервер остановлен")
	}
}
