package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

var L *slog.Logger

func Init(level string) {
	var lv slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lv = slog.LevelDebug
	case "warn":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	if strings.EqualFold(os.Getenv("APP_ENV"), "production") && lv == slog.LevelDebug {
		lv = slog.LevelInfo
	}
	L = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lv}))
	slog.SetDefault(L)
}

func Writer() io.Writer { return os.Stdout }
