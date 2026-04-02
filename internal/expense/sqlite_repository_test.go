package expense_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/ArtroxGabriel/accounter/internal/expense"
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

func setupTestRepo(t *testing.T, db *sql.DB) expense.Repository {
	t.Helper()
	injector := do.New()
	do.ProvideValue(injector, database.NewFromDB(db))
	repo, err := expense.NewSQLiteRepository(injector)
	require.NoError(t, err)
	return repo
}

// category setup helper.
func seedTestCategory(t *testing.T, db *sql.DB, name string) int64 {
	t.Helper()
	res, err := db.Exec("INSERT INTO categories (name, icon) VALUES (?, '📦')", name)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return id
}

func TestSQLiteRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := setupTestRepo(t, db)
	ctx := context.Background()

	// Seed some categories
	catFood := seedTestCategory(t, db, "AlimentaçãoTest")
	catTransport := seedTestCategory(t, db, "TransporteTest")

	t.Run("Create inserts expense with valid category", func(t *testing.T) {
		date := time.Now().UTC().Truncate(24 * time.Hour) // just day precision for tests
		input := expense.Expense{
			Amount:      15000,
			Description: "Groceries",
			CategoryID:  catFood,
			Date:        date,
		}

		created, err := repo.Create(ctx, input)
		require.NoError(t, err)
		assert.NotZero(t, created.ID)
		assert.Equal(t, int64(15000), created.Amount)
		assert.Equal(t, "Groceries", created.Description)
		assert.Equal(t, catFood, created.CategoryID)
		assert.Equal(t, "AlimentaçãoTest", created.Category)
		assert.Equal(t, date.Format(time.DateOnly), created.Date.Format(time.DateOnly))
		assert.NotZero(t, created.CreatedAt)
	})

	t.Run("Create with invalid category_id returns FK constraint error", func(t *testing.T) {
		input := expense.Expense{
			Amount:      1000,
			Description: "Invalid",
			CategoryID:  999999,
			Date:        time.Now(),
		}
		_, err := repo.Create(ctx, input)
		require.Error(t, err)
	})

	t.Run("GetByID returns expense or ErrNotFound", func(t *testing.T) {
		input := expense.Expense{
			Amount:      5000,
			Description: "Lunch",
			CategoryID:  catFood,
			Date:        time.Now().UTC(),
		}
		created, err := repo.Create(ctx, input)
		require.NoError(t, err)

		fetched, getErr := repo.GetByID(ctx, created.ID)
		require.NoError(t, getErr)
		assert.Equal(t, created.ID, fetched.ID)
		assert.Equal(t, "AlimentaçãoTest", fetched.Category)

		_, notFoundErr := repo.GetByID(ctx, 999999)
		require.ErrorIs(t, notFoundErr, expense.ErrNotFound)
	})

	t.Run("List with filtering and pagination", func(t *testing.T) {
		// Clean expenses for precise list tests
		_, _ = db.Exec("DELETE FROM expenses")

		now := time.Now().UTC()
		date1 := now.AddDate(0, 0, -10).Format(time.DateOnly)
		date2 := now.AddDate(0, 0, -5).Format(time.DateOnly)
		dateNew := now.Format(time.DateOnly)

		_, _ = db.Exec("INSERT INTO expenses (amount, description, category_id, date) VALUES (?, ?, ?, ?)",
			1000, "Old Food", catFood, date1)
		_, _ = db.Exec("INSERT INTO expenses (amount, description, category_id, date) VALUES (?, ?, ?, ?)",
			2000, "Mid Transport", catTransport, date2)
		_, _ = db.Exec("INSERT INTO expenses (amount, description, category_id, date) VALUES (?, ?, ?, ?)",
			3000, "New Food", catFood, dateNew)

		// 10.5 Date range filter
		from, _ := time.Parse(time.DateOnly, date2)
		to := time.Now().UTC().AddDate(0, 0, 1) // Tomorrow
		res, err := repo.List(ctx, expense.ListFilter{From: from, To: to})
		require.NoError(t, err)
		assert.Len(t, res, 2) // Mid and New

		// 10.8 List ordered by date descending (newest first)
		assert.Equal(t, "New Food", res[0].Description)
		assert.Equal(t, "Mid Transport", res[1].Description)

		// 10.6 Category filter
		resCat, err := repo.List(ctx, expense.ListFilter{Category: &catFood})
		require.NoError(t, err)
		assert.Len(t, resCat, 2)
		assert.True(t, resCat[0].CategoryID == catFood && resCat[1].CategoryID == catFood)

		// 10.7 Pagination
		resLimit1, err := repo.List(ctx, expense.ListFilter{Limit: 1, Offset: 1})
		require.NoError(t, err)
		assert.Len(t, resLimit1, 1)
	})

	t.Run("Delete removes expense", func(t *testing.T) {
		input := expense.Expense{
			Amount:      1000,
			Description: "To Delete",
			CategoryID:  catFood,
			Date:        time.Now(),
		}
		created, err := repo.Create(ctx, input)
		require.NoError(t, err)

		delErr := repo.Delete(ctx, created.ID)
		require.NoError(t, delErr)

		_, getErr := repo.GetByID(ctx, created.ID)
		require.ErrorIs(t, getErr, expense.ErrNotFound)

		delAgainErr := repo.Delete(ctx, created.ID)
		require.ErrorIs(t, delAgainErr, expense.ErrNotFound)
	})

	t.Run("Summary aggregated data", func(t *testing.T) {
		_, _ = db.Exec("DELETE FROM expenses") // Clean

		now := time.Now().UTC()
		date1 := now.Format(time.RFC3339)

		// Total Food: 4000, Total Transport: 2000
		_, _ = repo.Create(ctx, expense.Expense{Amount: 1000, CategoryID: catFood, Date: now})
		_, _ = repo.Create(ctx, expense.Expense{Amount: 3000, CategoryID: catFood, Date: now})
		_, _ = repo.Create(ctx, expense.Expense{Amount: 2000, CategoryID: catTransport, Date: now})

		t1, _ := time.Parse(time.RFC3339, date1)
		summary, err := repo.Summary(ctx, t1.AddDate(0, 0, -1), t1.AddDate(0, 0, 1))
		require.NoError(t, err)

		assert.Equal(t, int64(6000), summary.Total)
		assert.Equal(t, 3, summary.ExpenseCount)
		assert.Len(t, summary.ByCategory, 2)

		// Order might vary depending on SQLite GROUP BY. Let's find each.
		var foodTotal, transTotal int64
		for _, bc := range summary.ByCategory {
			switch bc.CategoryID {
			case catFood:
				foodTotal = bc.Total
			case catTransport:
				transTotal = bc.Total
			}
		}
		assert.Equal(t, int64(4000), foodTotal)
		assert.Equal(t, int64(2000), transTotal)
	})

	t.Run("Summary without expenses", func(t *testing.T) {
		_, _ = db.Exec("DELETE FROM expenses") // Clean

		summary, err := repo.Summary(ctx, time.Now().AddDate(-1, 0, 0), time.Now().AddDate(-1, 0, 1))
		require.NoError(t, err)
		assert.Equal(t, int64(0), summary.Total)
		assert.Equal(t, 0, summary.ExpenseCount)
		assert.Empty(t, summary.ByCategory)
	})
}
