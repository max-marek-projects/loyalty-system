// Package logger provides a global zap.Logger instance with configurable log level.
package logger

import (
	"fmt"

	"go.uber.org/zap"
)

// Log is the global logger instance. Initially a no-op logger.
var Log *zap.Logger = zap.NewNop()

// Initialize sets up the global logger with the specified log level.
// Parameters:
//   - level: log level string (DEBUG, INFO, WARNING, ERROR, FATAL).
//
// Returns an error if parsing the level or building the logger fails.
func Initialize(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	cfg := zap.NewProductionConfig()
	cfg.Level = lvl
	zl, err := cfg.Build()
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	Log = zl
	return nil
}
