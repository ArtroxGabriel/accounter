package category_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/ArtroxGabriel/accounter/internal/category"
	"github.com/ArtroxGabriel/accounter/internal/platform/database"
	"github.com/ArtroxGabriel/accounter/internal/platform/migrate"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, ":memory:")
	require.NoError(t, err)
	require.NoError(t, migrate.Run(ctx, db))
	t.Cleanup(func() { db.Close() })
	return db
}

func setupTestRepo(t *testing.T, db *sql.DB) category.Repository {
	t.Helper()
	injector := do.New()
	do.ProvideValue(injector, database.NewFromDB(db))
	repo, err := category.NewSQLiteRepository(injector)
	require.NoError(t, err)
	return repo
}

func TestSQLiteRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := setupTestRepo(t, db)

	ctx := context.Background()

	t.Run("Create and GetByID", func(t *testing.T) {
		input := category.Category{
			Name: "New Category",
			Icon: "🚀",
		}

		cat, createErr := repo.Create(ctx, input)
		require.NoError(t, createErr)
		assert.NotZero(t, cat.ID)
		assert.Equal(t, "New Category", cat.Name)
		assert.Equal(t, "🚀", cat.Icon)
		assert.NotZero(t, cat.CreatedAt)

		saved, getErr := repo.GetByID(ctx, cat.ID)
		require.NoError(t, getErr)
		assert.Equal(t, cat.ID, saved.ID)
		assert.Equal(t, cat.Name, saved.Name)
		assert.Equal(t, cat.Icon, saved.Icon)
	})

	t.Run("Create duplicate name returns error", func(t *testing.T) {
		input := category.Category{
			Name: "Duplicate",
			Icon: "🚀",
		}
		_, createErr1 := repo.Create(ctx, input)
		require.NoError(t, createErr1)

		_, createErr2 := repo.Create(ctx, input)
		require.Error(t, createErr2)
	})

	t.Run("GetByID not found", func(t *testing.T) {
		_, getErr := repo.GetByID(ctx, 99999)
		require.ErrorIs(t, getErr, category.ErrNotFound)
	})

	t.Run("GetByName", func(t *testing.T) {
		input := category.Category{
			Name: "ByNameTarget",
			Icon: "🎯",
		}
		cat, createErr := repo.Create(ctx, input)
		require.NoError(t, createErr)

		// Test exact match
		found, getErr1 := repo.GetByName(ctx, "ByNameTarget")
		require.NoError(t, getErr1)
		assert.Equal(t, cat.ID, found.ID)

		// Test case-insensitive match
		foundLower, getErr2 := repo.GetByName(ctx, "bynametarget")
		require.NoError(t, getErr2)
		assert.Equal(t, cat.ID, foundLower.ID)

		// Test not found
		_, getErr3 := repo.GetByName(ctx, "NotExists")
		require.ErrorIs(t, getErr3, category.ErrNotFound)
	})

	t.Run("Exists", func(t *testing.T) {
		input := category.Category{
			Name: "ExistsTarget",
			Icon: "🎯",
		}
		cat, createErr := repo.Create(ctx, input)
		require.NoError(t, createErr)

		exists, existsErr1 := repo.Exists(ctx, cat.ID)
		require.NoError(t, existsErr1)
		assert.True(t, exists)

		exists2, existsErr2 := repo.Exists(ctx, 99999)
		require.NoError(t, existsErr2)
		assert.False(t, exists2)
	})

	t.Run("List", func(t *testing.T) {
		categories, listErr := repo.List(ctx)
		require.NoError(t, listErr)
		assert.NotEmpty(t, categories)

		// Just ensure it's ordered by name.
		for i := range len(categories) - 1 {
			assert.LessOrEqual(t, categories[i].Name, categories[i+1].Name)
		}
	})

	t.Run("Update", func(t *testing.T) {
		input := category.Category{
			Name: "To Update",
			Icon: "📱",
		}
		cat, createErr := repo.Create(ctx, input)
		require.NoError(t, createErr)

		cat.Name = "Updated Name"
		cat.Icon = "💻"

		updated, updateErr := repo.Update(ctx, cat)
		require.NoError(t, updateErr)
		assert.Equal(t, "Updated Name", updated.Name)
		assert.Equal(t, "💻", updated.Icon)
	})

	t.Run("Delete", func(t *testing.T) {
		input := category.Category{
			Name: "To Delete",
			Icon: "🗑️",
		}
		cat, createErr := repo.Create(ctx, input)
		require.NoError(t, createErr)

		delErr := repo.Delete(ctx, cat.ID)
		require.NoError(t, delErr)

		_, getErr := repo.GetByID(ctx, cat.ID)
		require.ErrorIs(t, getErr, category.ErrNotFound)
	})

	t.Run("Delete with FK constraint returns error", func(t *testing.T) {
		input := category.Category{
			Name: "Has Expenses",
			Icon: "💰",
		}
		cat, createErr := repo.Create(ctx, input)
		require.NoError(t, createErr)

		// Insert fake expense to trigger FK
		_, execErr := db.ExecContext(ctx,
			"INSERT INTO expenses (amount, category_id, date) VALUES (1000, ?, date('now'))",
			cat.ID)
		require.NoError(t, execErr)

		delErr := repo.Delete(ctx, cat.ID)
		require.Error(t, delErr)
	})
}
