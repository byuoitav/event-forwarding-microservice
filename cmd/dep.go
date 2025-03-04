package main

import (
	"fmt"
	"log/slog"
)

func stringToLogLevel(lvl string) (slog.Level, error) {
	switch lvl {
	case "Info":
		slog.Debug("Setting level to Info")
		return slog.LevelInfo, nil
	case "Debug":
		slog.Debug("Setting level to Debug")
		return slog.LevelDebug, nil
	case "Warn":
		slog.Debug("Setting level to Warn")
		return slog.LevelWarn, nil
	case "Error":
		slog.Debug("Setting level to Error")
		return slog.LevelError, nil
	default:
		slog.Debug("Log level doesn't match an existing level, please use Info, Debug, Warn, or Error")
		return slog.LevelInfo, fmt.Errorf("log level doesn't match an existing level, please use Info, Debug, Warn, or Error")
	}
}
