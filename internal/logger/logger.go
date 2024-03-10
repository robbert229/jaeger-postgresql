package logger

import (
	"fmt"
	"log/slog"
	"os"
)

// New returns a new logger.
func New(loglevelStr *string) (*slog.Logger, error) {
	levelFn := func() (slog.Level, error) {
		if loglevelStr == nil {
			return slog.LevelWarn, nil
		}

		switch *loglevelStr {
		case "info":
			return slog.LevelInfo, nil
		case "warn":
			return slog.LevelWarn, nil
		case "error":
			return slog.LevelError, nil
		case "debug":
			return slog.LevelDebug, nil
		default:
			return 0, fmt.Errorf("invalid log level: %s", *loglevelStr)
		}
	}
	level, err := levelFn()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})), nil
}
