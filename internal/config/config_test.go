package config_test

import (
	"os"
	"testing"

	"github.com/ArtroxGabriel/accounter/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Test cases
	t.Run("fails when required vars are missing", func(t *testing.T) {
		os.Clearenv()
		_, err := config.Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing BEARER_TOKEN")
	})

	t.Run("fails when telegram token is missing", func(t *testing.T) {
		os.Clearenv()
		t.Setenv("BEARER_TOKEN", "test-bearer")
		_, err := config.Load()
		require.ErrorIs(t, err, config.ErrMissingTelegramToken)
	})

	t.Run("loads with defaults", func(t *testing.T) {
		os.Clearenv()
		t.Setenv("BEARER_TOKEN", "test-bearer")
		t.Setenv("TELEGRAM_TOKEN", "test-telegram")

		cfg, err := config.Load()
		require.NoError(t, err)

		assert.Equal(t, "test-bearer", cfg.BearerToken)
		assert.Equal(t, "test-telegram", cfg.TelegramToken)
		assert.Equal(t, "8080", cfg.Port)
		assert.Equal(t, "accounter.db", cfg.DatabasePath)
		assert.Equal(t, "info", cfg.LogLevel)
		assert.Equal(t, "development", cfg.Environment)
		assert.Equal(t, "America/Sao_Paulo", cfg.Timezone)
	})

	t.Run("overrides defaults with environment variables", func(t *testing.T) {
		os.Clearenv()
		t.Setenv("BEARER_TOKEN", "test-bearer")
		t.Setenv("TELEGRAM_TOKEN", "test-telegram")
		t.Setenv("PORT", "9000")
		t.Setenv("DATABASE_PATH", "/tmp/test.db")
		t.Setenv("LOG_LEVEL", "debug")
		t.Setenv("ENVIRONMENT", "production")
		t.Setenv("TIMEZONE", "UTC")

		cfg, err := config.Load()
		require.NoError(t, err)

		assert.Equal(t, "9000", cfg.Port)
		assert.Equal(t, "/tmp/test.db", cfg.DatabasePath)
		assert.Equal(t, "debug", cfg.LogLevel)
		assert.Equal(t, "production", cfg.Environment)
		assert.Equal(t, "UTC", cfg.Timezone)
	})
}
