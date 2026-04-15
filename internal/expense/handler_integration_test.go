//go:build integration

package expense_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/ArtroxGabriel/accounter/internal/category"
	"github.com/ArtroxGabriel/accounter/internal/expense"
	"github.com/ArtroxGabriel/accounter/internal/platform/database"
	"github.com/ArtroxGabriel/accounter/internal/platform/migrate"
	"github.com/go-chi/chi/v5"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupExpenseIntegrationRouter(t *testing.T) (http.Handler, *sql.DB) {
	t.Helper()

	ctx := context.Background()
	db, err := database.Open(ctx, ":memory:")
	require.NoError(t, err)
	require.NoError(t, migrate.Run(ctx, db))
	t.Cleanup(func() { _ = db.Close() })

	injector := do.New()
	do.ProvideValue(injector, database.NewFromDB(db))

	categoryRepo, err := category.NewSQLiteRepository(injector)
	require.NoError(t, err)
	do.ProvideValue(injector, categoryRepo)

	categorySvc, err := category.NewService(injector)
	require.NoError(t, err)
	do.ProvideValue(injector, categorySvc)
	do.ProvideValue[expense.CategoryChecker](injector, categorySvc)

	expenseRepo, err := expense.NewSQLiteRepository(injector)
	require.NoError(t, err)
	do.ProvideValue(injector, expenseRepo)

	expenseSvc, err := expense.NewService(injector)
	require.NoError(t, err)
	do.ProvideValue(injector, expenseSvc)

	h, err := expense.NewHandler(injector)
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Route("/api/expenses", h.Routes)

	return r, db
}

func seedCategory(t *testing.T, db *sql.DB, name string) int64 {
	t.Helper()

	res, err := db.ExecContext(context.Background(),
		"INSERT INTO categories (name, icon) VALUES (?, ?)",
		name, "📦",
	)
	require.NoError(t, err)

	id, err := res.LastInsertId()
	require.NoError(t, err)

	return id
}

func TestExpenseHandlerIntegration_CRUDAndSummary(t *testing.T) {
	router, db := setupExpenseIntegrationRouter(t)
	categoryID := seedCategory(t, db, "Food")

	input := expense.CreateExpenseInput{
		Amount:      2550,
		Description: "Lunch",
		CategoryID:  categoryID,
		Date:        time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC),
	}
	body, err := json.Marshal(input)
	require.NoError(t, err)

	createReq, err := http.NewRequest(http.MethodPost, "/api/expenses", bytes.NewReader(body))
	require.NoError(t, err)
	createReq.Header.Set("Content-Type", "application/json")

	createRes := httptest.NewRecorder()
	router.ServeHTTP(createRes, createReq)

	require.Equal(t, http.StatusCreated, createRes.Code)

	var created expense.Expense
	require.NoError(t, json.Unmarshal(createRes.Body.Bytes(), &created))
	assert.NotZero(t, created.ID)
	assert.Equal(t, int64(2550), created.Amount)
	assert.Equal(t, categoryID, created.CategoryID)

	getReq, err := http.NewRequest(http.MethodGet, "/api/expenses/"+strconv.FormatInt(created.ID, 10), nil)
	require.NoError(t, err)

	getRes := httptest.NewRecorder()
	router.ServeHTTP(getRes, getReq)

	require.Equal(t, http.StatusOK, getRes.Code)

	var fetched expense.Expense
	require.NoError(t, json.Unmarshal(getRes.Body.Bytes(), &fetched))
	assert.Equal(t, created.ID, fetched.ID)
	assert.Equal(t, "Food", fetched.Category)

	listReq, err := http.NewRequest(http.MethodGet, "/api/expenses?limit=10", nil)
	require.NoError(t, err)

	listRes := httptest.NewRecorder()
	router.ServeHTTP(listRes, listReq)

	require.Equal(t, http.StatusOK, listRes.Code)

	var listed []expense.Expense
	require.NoError(t, json.Unmarshal(listRes.Body.Bytes(), &listed))
	require.NotEmpty(t, listed)
	assert.Equal(t, created.ID, listed[0].ID)

	summaryReq, err := http.NewRequest(http.MethodGet, "/api/expenses/summary?from=2025-01-01&to=2025-02-01", nil)
	require.NoError(t, err)

	summaryRes := httptest.NewRecorder()
	router.ServeHTTP(summaryRes, summaryReq)

	require.Equal(t, http.StatusOK, summaryRes.Code)

	var summary expense.Summary
	require.NoError(t, json.Unmarshal(summaryRes.Body.Bytes(), &summary))
	assert.Equal(t, int64(2550), summary.Total)
	assert.Equal(t, 1, summary.ExpenseCount)

	deleteReq, err := http.NewRequest(http.MethodDelete, "/api/expenses/"+strconv.FormatInt(created.ID, 10), nil)
	require.NoError(t, err)

	deleteRes := httptest.NewRecorder()
	router.ServeHTTP(deleteRes, deleteReq)

	require.Equal(t, http.StatusNoContent, deleteRes.Code)
}

func TestExpenseHandlerIntegration_CreateWithInvalidCategory(t *testing.T) {
	router, _ := setupExpenseIntegrationRouter(t)

	input := expense.CreateExpenseInput{Amount: 1000, CategoryID: 999999, Description: "Invalid category"}
	body, err := json.Marshal(input)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/api/expenses", bytes.NewReader(body))
	require.NoError(t, err)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusBadRequest, res.Code)
}

func TestExpenseHandlerIntegration_SummaryValidation(t *testing.T) {
	router, _ := setupExpenseIntegrationRouter(t)

	tests := []struct {
		name string
		url  string
	}{
		{name: "missing query params", url: "/api/expenses/summary"},
		{name: "invalid from date", url: "/api/expenses/summary?from=01-01-2025&to=2025-02-01"},
		{name: "invalid to date", url: "/api/expenses/summary?from=2025-01-01&to=31-01-2025"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, tt.url, nil)
			require.NoError(t, err)

			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)

			assert.Equal(t, http.StatusBadRequest, res.Code)
		})
	}
}
