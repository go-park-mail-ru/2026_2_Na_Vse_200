package logger

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/rs/zerolog"
)

func New(
	out io.Writer,
	level string,
	format string,
) (zerolog.Logger, error) {
	level = strings.ToLower(strings.TrimSpace(level))
	format = strings.ToLower(strings.TrimSpace(format))

	if level == "" {
		level = "info"
	}
	if format == "" {
		format = "json"
	}
	parsedLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		return zerolog.Logger{}, fmt.Errorf(
			"уровень логирования %q: %w",
			level,
			err,
		)
	}

	var writer io.Writer = out

	switch format {
	case "json":
	case "console":
		writer = zerolog.ConsoleWriter{
			Out:     out,
			NoColor: true,
		}
	default:
		return zerolog.Logger{}, fmt.Errorf(
			"неизвестный формат логирования %q",
			format,
		)
	}

	logger := zerolog.New(writer).
		Level(parsedLevel).
		With().
		Timestamp().
		Str("service", "music-api").
		Logger()

	return logger, nil
}

type ctxKey struct{}

// WithRequestID кладёт идентификатор запроса в контекст.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// RequestID возвращает идентификатор запроса из контекста или пустую строку.
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

// UsecaseStarted отмечает вход в сценарий с полями логгера текущего запроса.
func UsecaseStarted(ctx context.Context, operation string) {
	zerolog.Ctx(ctx).Debug().
		Str("layer", "usecase").
		Str("operation", operation).
		Str("phase", "start").
		Msg("начало сценария")
}
