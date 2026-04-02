package migrate_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/ArtroxGabriel/accounter/internal/platform/migrate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func createTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	// Enable foreign keys for the test connection.
	_, err = db.Exec("PRAGMA foreign_keys=ON;")
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Close()
	})
	return db
}

func TestRun(t *testing.T) {
	ctx := context.Background()

	t.Run("applies migrations successfully", func(t *testing.T) {
		db := createTestDB(t)

		err := migrate.Run(ctx, db)
		require.NoError(t, err)

		// Check if tables exist
		tables := []string{"categories", "expenses", "schema_migrations"}
		for _, table := range tables {
			var exists bool
			err = db.
				QueryRow("SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name=?)", table).
				Scan(&exists)
			require.NoError(t, err)
			assert.True(t, exists, "Table %s should exist", table)
		}

		// Check categories seed
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 9, count, "Should have 9 categories")

		// Check schema migrations count
		var migrationsCount int
		err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&migrationsCount)
		require.NoError(t, err)
		assert.Equal(t, 3, migrationsCount, "Should have 3 migrations recorded")
	})

	t.Run("is idempotent", func(t *testing.T) {
		db := createTestDB(t)

		// First run
		require.NoError(t, migrate.Run(ctx, db))

		// Second run should skip everything
		require.NoError(t, migrate.Run(ctx, db))

		var migrationsCount int
		err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&migrationsCount)
		require.NoError(t, err)
		assert.Equal(t, 3, migrationsCount)
	})

	t.Run("rolls back on error", func(_ *testing.T) {
		// This is a placeholder for rollback testing.
	})
}
