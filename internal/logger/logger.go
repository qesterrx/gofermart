package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Logger - кастомная структура для логирования
// Новый логгер создается через вызов NewLogger
type Logger struct {
	writer io.Writer
	log    zerolog.Logger
	with   string
}

// NewLogger - Возвращает новый экземпляр Logger
// На вход принимает:
// mode string - уровень логгера debug/info/error
// writer io.Writer - опционально - райтер для записи логов
func NewLogger(mode string, writer io.Writer) *Logger {
	io := writer
	if io == nil {
		io = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	}
	log := zerolog.New(io).With().Timestamp().Logger()

	switch mode {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	}

	return &Logger{log: log, with: "", writer: io}
}

// Debug - Запись в лог уровня DEBUG
func (l *Logger) Debug(msg string) {
	l.log.Debug().Msg(msg)
}

// Info - Запись в лог уровня INFO
func (l *Logger) Info(msg string) {
	l.log.Info().Msg(msg)
}

// Error - Запись в лог уровня ERROR
func (l *Logger) Error(msg string) {
	l.log.Error().Msg(msg)
}

// With - Создание нового логгера на основе существующего
// на вход принимает строку with string
func (l *Logger) With(with string) *Logger {
	with = l.with + "->" + with
	log := zerolog.New(l.writer).With().Timestamp().Logger().With().Str("with", with).Logger()
	return &Logger{log: log, with: with}
}
