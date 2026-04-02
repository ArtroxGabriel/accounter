package expense

import (
	"context"
	"errors"
	"time"

	"github.com/ArtroxGabriel/accounter/internal/platform/repository"
)

var ErrNotFound = errors.New("not found")

// Repository provides access to the expense storage.
type Repository interface {
	repository.Base[Expense, Expense]
	List(ctx context.Context, filter ListFilter) ([]Expense, error)
	Summary(ctx context.Context, from, to time.Time) (Summary, error)
}
