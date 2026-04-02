package logger_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/ArtroxGabriel/accounter/internal/platform/logger"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("creates a text handler in development", func(t *testing.T) {
		t.Parallel()
		l := logger.New("info", "development")
		assert.NotNil(t, l)
	})

	t.Run("creates a JSON handler in production", func(t *testing.T) {
		t.Parallel()
		l := logger.New("info", "production")
		assert.NotNil(t, l)
	})

	t.Run("parsers log levels correctly", func(t *testing.T) {
		t.Parallel()
		testCases := []struct {
			name     string
			input    string
			expected slog.Level
		}{
			{"debug", "debug", slog.LevelDebug},
			{"info", "info", slog.LevelInfo},
			{"warn", "warn", slog.LevelWarn},
			{"error", "error", slog.LevelError},
			{"unknown defaults to info", "something", slog.LevelInfo},
			{"case insensitive", "DEBUG", slog.LevelDebug},
		}

		ctx := context.Background()
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				l := logger.New(tc.input, "development")
				handler := l.Handler()
				assert.True(t, handler.Enabled(ctx, tc.expected))
				if tc.expected > slog.LevelDebug {
					assert.False(t, handler.Enabled(ctx, tc.expected-1))
				}
			})
		}
	})
}
