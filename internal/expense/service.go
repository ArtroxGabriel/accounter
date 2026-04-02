package expense

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/do/v2"
)

var (
	ErrInvalidInput     = errors.New("invalid input")
	ErrCategoryNotFound = errors.New("category not found")
)

// Service provides business logic for expenses.
type Service interface {
	Create(ctx context.Context, input CreateExpenseInput) (Expense, error)
	GetByID(ctx context.Context, id int64) (Expense, error)
	List(ctx context.Context, filter ListFilter) ([]Expense, error)
	Delete(ctx context.Context, id int64) error
	Summary(ctx context.Context, from, to time.Time) (Summary, error)
}

type DefaultService struct {
	repo    Repository
	checker CategoryChecker
}

func NewService(i do.Injector) (Service, error) {
	repo := do.MustInvoke[Repository](i)
	checker := do.MustInvoke[CategoryChecker](i)
	return &DefaultService{
		repo:    repo,
		checker: checker,
	}, nil
}

func (s *DefaultService) Create(ctx context.Context, input CreateExpenseInput) (Expense, error) {
	if input.Amount <= 0 {
		return Expense{}, fmt.Errorf("%w: amount must be greater than zero", ErrInvalidInput)
	}

	exists, err := s.checker.Exists(ctx, input.CategoryID)
	if err != nil {
		return Expense{}, fmt.Errorf("checking category: %w", err)
	}
	if !exists {
		return Expense{}, ErrCategoryNotFound
	}

	if input.Date.IsZero() {
		input.Date = time.Now()
	}

	return s.repo.Create(ctx, Expense{
		Amount:      input.Amount,
		Description: input.Description,
		CategoryID:  input.CategoryID,
		Date:        input.Date,
	})
}

func (s *DefaultService) GetByID(ctx context.Context, id int64) (Expense, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *DefaultService) List(ctx context.Context, filter ListFilter) ([]Expense, error) {
	return s.repo.List(ctx, filter)
}

func (s *DefaultService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *DefaultService) Summary(ctx context.Context, from, to time.Time) (Summary, error) {
	return s.repo.Summary(ctx, from, to)
}
