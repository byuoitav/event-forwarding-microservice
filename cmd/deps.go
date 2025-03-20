package main

import (
	"fmt"
	"log/slog"
	"strings"
)

func stringToLogLevel(lvl string) (slog.Level, error) {
	lvl = strings.ToLower(lvl)
	switch lvl {
	case "info":
		slog.Debug("Setting level to Info")
		return slog.LevelInfo, nil
	case "debug":
		slog.Debug("Setting level to Debug")
		return slog.LevelDebug, nil
	case "warn":
		slog.Debug("Setting level to Warn")
		return slog.LevelWarn, nil
	case "error":
		slog.Debug("Setting level to Error")
		return slog.LevelError, nil
	default:
		slog.Debug("Log level doesn't match an existing level, please use Info, Debug, Warn, or Error")
		return slog.LevelInfo, fmt.Errorf("log level doesn't match an existing level, please use Info, Debug, Warn, or Error")
	}
}
