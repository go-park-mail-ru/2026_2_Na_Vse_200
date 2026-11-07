package logger

import (
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
