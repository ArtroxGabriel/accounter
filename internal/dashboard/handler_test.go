package dashboard_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ArtroxGabriel/accounter/internal/category"
	"github.com/ArtroxGabriel/accounter/internal/config"
	"github.com/ArtroxGabriel/accounter/internal/dashboard"
	"github.com/ArtroxGabriel/accounter/internal/expense"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockExpenseService struct {
	createCalled bool
	listCalled   bool
	createErr    error
	listErr      error
	summaryErr   error
}

func (m *mockExpenseService) Create(_ context.Context, _ expense.CreateExpenseInput) (expense.Expense, error) {
	m.createCalled = true
	if m.createErr != nil {
		return expense.Expense{}, m.createErr
	}

	return expense.Expense{ID: 1, Amount: 1000, Description: "Lunch", CategoryID: 1, Date: time.Now()}, nil
}

func (m *mockExpenseService) GetByID(_ context.Context, _ int64) (expense.Expense, error) {
	return expense.Expense{}, nil
}

func (m *mockExpenseService) List(_ context.Context, _ expense.ListFilter) ([]expense.Expense, error) {
	m.listCalled = true
	if m.listErr != nil {
		return nil, m.listErr
	}

	return []expense.Expense{}, nil
}

func (m *mockExpenseService) Delete(_ context.Context, _ int64) error {
	return nil
}

func (m *mockExpenseService) Summary(_ context.Context, _, _ time.Time) (expense.Summary, error) {
	if m.summaryErr != nil {
		return expense.Summary{}, m.summaryErr
	}

	return expense.Summary{}, nil
}

type mockCategoryService struct {
	listErr error
}

func setupTestHandler(
	t *testing.T,
	expenseSvc expense.Service,
	categorySvc category.Service,
) *dashboard.Handler {
	t.Helper()

	injector := do.New()
	do.ProvideValue[config.Config](injector, config.Config{Timezone: "UTC"})
	do.ProvideValue[expense.Service](injector, expenseSvc)
	do.ProvideValue[category.Service](injector, categorySvc)
	do.ProvideValue[*slog.Logger](injector, slog.New(slog.NewTextHandler(io.Discard, nil)))

	h, err := dashboard.NewHandler(injector)
	require.NoError(t, err)

	return h
}

func (m *mockCategoryService) Create(_ context.Context, _ category.CreateCategoryInput) (category.Category, error) {
	return category.Category{}, nil
}

func (m *mockCategoryService) GetByID(_ context.Context, _ int64) (category.Category, error) {
	return category.Category{}, nil
}

func (m *mockCategoryService) GetByName(_ context.Context, _ string) (category.Category, error) {
	return category.Category{}, nil
}

func (m *mockCategoryService) List(_ context.Context) ([]category.Category, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}

	return []category.Category{}, nil
}

func (m *mockCategoryService) Exists(_ context.Context, _ int64) (bool, error) {
	return true, nil
}

func (m *mockCategoryService) Update(
	_ context.Context,
	_ int64,
	_ category.UpdateCategoryInput,
) (category.Category, error) {
	return category.Category{}, nil
}

func (m *mockCategoryService) Delete(_ context.Context, _ int64) error {
	return nil
}

func TestExpenseList_InvalidPeriodReturnsBadRequest(t *testing.T) {
	t.Parallel()

	h := setupTestHandler(t, &mockExpenseService{}, &mockCategoryService{})

	req := httptest.NewRequest(http.MethodGet, "/dashboard/expenses?period=invalid", nil)
	rr := httptest.NewRecorder()

	h.ExpenseList(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateExpense_InvalidAmountReturnsBadRequestAndSkipsService(t *testing.T) {
	t.Parallel()

	expenseSvc := &mockExpenseService{}
	h := setupTestHandler(t, expenseSvc, &mockCategoryService{})

	values := url.Values{}
	values.Set("amount", "abc")
	values.Set("category_id", "1")

	req := httptest.NewRequest(http.MethodPost, "/dashboard/expenses", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.CreateExpense(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.False(t, expenseSvc.createCalled)
}

func TestIndex_CategoryServiceErrorReturnsInternalServerError(t *testing.T) {
	t.Parallel()

	h := setupTestHandler(t, &mockExpenseService{}, &mockCategoryService{listErr: errors.New("boom")})

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rr := httptest.NewRecorder()

	h.Index(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestExpenseList_UsesDateFilterWhenPeriodIsMonth(t *testing.T) {
	t.Parallel()

	expenseSvc := &mockExpenseService{}
	h := setupTestHandler(t, expenseSvc, &mockCategoryService{})

	req := httptest.NewRequest(http.MethodGet, "/dashboard/expenses?period=month", nil)
	req.Header.Set("Hx-Request", "true")
	rr := httptest.NewRecorder()

	h.ExpenseList(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, expenseSvc.listCalled)
}

func TestNewHandler_InvalidTimezoneReturnsError(t *testing.T) {
	t.Parallel()

	injector := do.New()
	do.ProvideValue[config.Config](injector, config.Config{Timezone: "not/a-timezone"})
	do.ProvideValue[expense.Service](injector, &mockExpenseService{})
	do.ProvideValue[category.Service](injector, &mockCategoryService{})
	do.ProvideValue[*slog.Logger](injector, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := dashboard.NewHandler(injector)
	require.Error(t, err)
	assert.ErrorContains(t, err, fmt.Sprintf("loading timezone %s", "not/a-timezone"))
}
