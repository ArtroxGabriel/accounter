package expense_test

import (
	"context"
	"testing"
	"time"

	"github.com/ArtroxGabriel/accounter/internal/expense"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, input expense.Expense) (expense.Expense, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(expense.Expense), args.Error(1)
}

func (m *MockRepository) GetByID(ctx context.Context, id int64) (expense.Expense, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(expense.Expense), args.Error(1)
}

func (m *MockRepository) List(ctx context.Context, filter expense.ListFilter) ([]expense.Expense, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]expense.Expense), args.Error(1)
}

func (m *MockRepository) Delete(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) Summary(ctx context.Context, from, to time.Time) (expense.Summary, error) {
	args := m.Called(ctx, from, to)
	return args.Get(0).(expense.Summary), args.Error(1)
}

type MockCategoryChecker struct {
	mock.Mock
}

func (m *MockCategoryChecker) Exists(ctx context.Context, id int64) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

func setupTestService(t *testing.T, repo expense.Repository, checker expense.CategoryChecker) expense.Service {
	t.Helper()
	injector := do.New()
	do.ProvideValue(injector, repo)
	do.ProvideValue(injector, checker)
	svc, err := expense.NewService(injector)
	require.NoError(t, err)
	return svc
}

func TestDefaultService_Create(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		input      expense.CreateExpenseInput
		setupMock  func(*MockRepository, *MockCategoryChecker)
		wantResult expense.Expense
		wantErr    error
	}{
		{
			name: "Create with valid input delegates to repo",
			input: expense.CreateExpenseInput{
				Amount:      1000,
				Description: "Lunch",
				CategoryID:  1,
				Date:        time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC),
			},
			setupMock: func(repo *MockRepository, checker *MockCategoryChecker) {
				checker.On("Exists", ctx, int64(1)).Return(true, nil)
				repo.On("Create", ctx, expense.Expense{
					Amount:      1000,
					Description: "Lunch",
					CategoryID:  1,
					Date:        time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC),
				}).Return(expense.Expense{ID: 1, Amount: 1000, Description: "Lunch"}, nil)
			},
			wantResult: expense.Expense{ID: 1, Amount: 1000, Description: "Lunch"},
		},
		{
			name: "Create with amount <= 0 returns validation error",
			input: expense.CreateExpenseInput{
				Amount:      0,
				Description: "Lunch",
				CategoryID:  1,
			},
			setupMock: func(_ *MockRepository, _ *MockCategoryChecker) {},
			wantErr:   expense.ErrInvalidInput,
		},
		{
			name: "Create with non-existent category returns error",
			input: expense.CreateExpenseInput{
				Amount:      1000,
				Description: "Lunch",
				CategoryID:  99,
				Date:        time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC),
			},
			setupMock: func(_ *MockRepository, checker *MockCategoryChecker) {
				checker.On("Exists", ctx, int64(99)).Return(false, nil)
			},
			wantErr: expense.ErrCategoryNotFound,
		},
		{
			name: "Create with zero date defaults to today",
			input: expense.CreateExpenseInput{
				Amount:      1000,
				Description: "Lunch",
				CategoryID:  1,
				// Date left zero
			},
			setupMock: func(repo *MockRepository, checker *MockCategoryChecker) {
				checker.On("Exists", ctx, int64(1)).Return(true, nil)
				// Expect repo to be called with some non-zero date
				repo.On("Create", ctx, mock.MatchedBy(func(in expense.Expense) bool {
					return !in.Date.IsZero() && in.Amount == 1000
				})).Return(expense.Expense{ID: 1, Amount: 1000}, nil)
			},
			wantResult: expense.Expense{ID: 1, Amount: 1000},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockRepository)
			checker := new(MockCategoryChecker)
			tt.setupMock(repo, checker)

			svc := setupTestService(t, repo, checker)
			got, err := svc.Create(ctx, tt.input)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantResult.ID, got.ID)
			repo.AssertExpectations(t)
			checker.AssertExpectations(t)
		})
	}
}

func TestDefaultService_List(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	checker := new(MockCategoryChecker)
	svc := setupTestService(t, repo, checker)

	filter := expense.ListFilter{Limit: 10}
	expected := []expense.Expense{{ID: 1, Amount: 100}}
	repo.On("List", ctx, filter).Return(expected, nil)

	got, err := svc.List(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, expected, got)
	repo.AssertExpectations(t)
}

func TestDefaultService_Summary(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	checker := new(MockCategoryChecker)
	svc := setupTestService(t, repo, checker)

	from := time.Now().AddDate(0, 0, -1)
	to := time.Now()
	expected := expense.Summary{Total: 100}
	repo.On("Summary", ctx, from, to).Return(expected, nil)

	got, err := svc.Summary(ctx, from, to)
	require.NoError(t, err)
	assert.Equal(t, expected, got)
	repo.AssertExpectations(t)
}

func TestDefaultService_Delete(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	checker := new(MockCategoryChecker)
	svc := setupTestService(t, repo, checker)

	repo.On("Delete", ctx, int64(1)).Return(nil)

	err := svc.Delete(ctx, 1)
	require.NoError(t, err)
	repo.AssertExpectations(t)
}
