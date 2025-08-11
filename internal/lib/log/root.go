package log

import (
	"log/slog"
	"os"

	"github.com/mistweaverco/zana-client/internal/lib/version"
)

var logLevel slog.Level = slog.LevelDebug

func SetLogLevel(level slog.Level) {
	slog.SetLogLoggerLevel(level)
}

func NewLogger() *slog.Logger {
	// When running in a production environment,
	// set the log level to Error
	if version.VERSION != "" {
		logLevel = slog.LevelError
	}
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))
}