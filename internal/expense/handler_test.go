package expense_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ArtroxGabriel/accounter/internal/expense"
	"github.com/go-chi/chi/v5"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) Create(ctx context.Context, input expense.CreateExpenseInput) (expense.Expense, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(expense.Expense), args.Error(1)
}

func (m *MockService) GetByID(ctx context.Context, id int64) (expense.Expense, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(expense.Expense), args.Error(1)
}

func (m *MockService) List(ctx context.Context, filter expense.ListFilter) ([]expense.Expense, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]expense.Expense), args.Error(1)
}

func (m *MockService) Delete(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockService) Summary(ctx context.Context, from, to time.Time) (expense.Summary, error) {
	args := m.Called(ctx, from, to)
	return args.Get(0).(expense.Summary), args.Error(1)
}

func setupTestHandler(t *testing.T, s expense.Service) *expense.Handler {
	t.Helper()
	injector := do.New()
	do.ProvideValue(injector, s)
	h, err := expense.NewHandler(injector)
	require.NoError(t, err)
	return h
}

func TestHandler_Create(t *testing.T) {
	t.Run("valid input returns 201", func(t *testing.T) {
		svc := new(MockService)
		h := setupTestHandler(t, svc)

		input := expense.CreateExpenseInput{Amount: 1000, CategoryID: 1, Description: "Lunch"}
		expected := expense.Expense{ID: 1, Amount: 1000, CategoryID: 1, Description: "Lunch"}
		svc.On("Create", mock.Anything, mock.MatchedBy(func(in expense.CreateExpenseInput) bool {
			return in.Amount == input.Amount && in.CategoryID == input.CategoryID
		})).Return(expected, nil)

		body, _ := json.Marshal(input)
		req, _ := http.NewRequest(http.MethodPost, "/api/expenses", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		h.Create(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		svc.AssertExpectations(t)
	})
}

func TestHandler_List(t *testing.T) {
	svc := new(MockService)
	h := setupTestHandler(t, svc)

	svc.On("List", mock.Anything, mock.MatchedBy(func(f expense.ListFilter) bool {
		return f.Limit == 10
	})).Return([]expense.Expense{}, nil)

	req, _ := http.NewRequest(http.MethodGet, "/api/expenses?limit=10", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandler_Summary(t *testing.T) {
	svc := new(MockService)
	h := setupTestHandler(t, svc)

	svc.On("Summary", mock.Anything, mock.Anything, mock.Anything).Return(expense.Summary{}, nil)

	req, _ := http.NewRequest(http.MethodGet, "/api/expenses/summary?from=2024-01-01&to=2024-01-31", nil)
	rr := httptest.NewRecorder()

	h.Summary(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandler_Delete(t *testing.T) {
	svc := new(MockService)
	h := setupTestHandler(t, svc)

	svc.On("Delete", mock.Anything, int64(1)).Return(nil)

	req, _ := http.NewRequest(http.MethodDelete, "/api/expenses/1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.Delete(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
}
