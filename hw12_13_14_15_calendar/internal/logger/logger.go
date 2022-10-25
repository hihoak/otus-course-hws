package logger

import (
	"os"
	"strings"

	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/app"
	"github.com/rs/zerolog"
)

type Logger struct {
	app.Logger
	logger zerolog.Logger
}

func convertToLogLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "error":
		return zerolog.ErrorLevel
	case "warn":
		return zerolog.WarnLevel
	case "info":
		return zerolog.InfoLevel
	case "debug":
		return zerolog.DebugLevel
	default:
		return zerolog.InfoLevel
	}
}

func New(level string) *Logger {
	return &Logger{
		logger: zerolog.New(os.Stdout).Level(convertToLogLevel(level)).With().Timestamp().Logger(),
	}
}

func (l Logger) Info() *zerolog.Event {
	return l.logger.Info()
}

func (l Logger) Error() *zerolog.Event {
	return l.logger.Error()
}

func (l Logger) Warn() *zerolog.Event {
	return l.logger.Warn()
}

func (l Logger) Debug() *zerolog.Event {
	return l.logger.Debug()
}

func (l Logger) Fatal() *zerolog.Event {
	return l.logger.Fatal()
}
