package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

var Log zerolog.Logger = zerolog.New(io.Discard)

type Logger struct {
	log  zerolog.Logger
	with string
}

func NewLogger(mode string, writer io.Writer) *Logger {
	io := writer
	if io == nil {
		io = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	}
	log := zerolog.New(io).With().Timestamp().Logger()
	return &Logger{log: log, with: ""}
}

func (l *Logger) Debug(msg string) {
	l.log.Debug().Msg(msg)
}

func (l *Logger) Info(msg string) {
	l.log.Info().Msg(msg)
}

func (l *Logger) Error(msg string) {
	l.log.Error().Msg(msg)
}

func (l *Logger) With(with string) *Logger {
	l.with = l.with + "->" + with
	log := l.log.With().Str("with", l.with).Logger()
	return &Logger{log: log, with: l.with}
}
