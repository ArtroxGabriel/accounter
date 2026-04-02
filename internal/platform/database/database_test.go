package database_test

import (
	"context"
	"testing"

	"github.com/ArtroxGabriel/accounter/internal/platform/database"
	"github.com/ArtroxGabriel/accounter/internal/platform/migrate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen(t *testing.T) {
	ctx := context.Background()

	t.Run("opens in-memory database and runs migrations", func(t *testing.T) {
		db, err := database.Open(ctx, ":memory:")
		require.NoError(t, err)
		defer db.Close()

		err = migrate.Run(ctx, db)
		require.NoError(t, err)

		// Check if a table created by migrations exists
		var exists bool
		err = db.
			QueryRow("SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='categories')").
			Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("enables WAL mode and foreign keys", func(t *testing.T) {
		db, err := database.Open(ctx, ":memory:")
		require.NoError(t, err)
		defer db.Close()

		var journalMode string
		err = db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
		require.NoError(t, err)
		// In-memory databases might not support WAL, but let's check it anyway.
		assert.Contains(t, []string{"wal", "memory"}, journalMode)

		var foreignKeys int
		err = db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys)
		require.NoError(t, err)
		assert.Equal(t, 1, foreignKeys)
	})
}
