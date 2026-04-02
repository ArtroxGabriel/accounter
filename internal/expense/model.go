package expense

import (
	"context"
	"time"
)

// Expense represents a single financial transaction.
type Expense struct {
	ID          int64     `json:"id"`
	Amount      int64     `json:"amount"` // Amount in cents (BRL)
	Description string    `json:"description"`
	CategoryID  int64     `json:"category_id"`
	Category    string    `json:"category"` // Denormalized name (read-only, from JOINs)
	Date        time.Time `json:"date"`     // Date of the expense
	CreatedAt   time.Time `json:"created_at"`
}

// CreateExpenseInput is the input for creating a new expense.
type CreateExpenseInput struct {
	Amount      int64     `json:"amount"` // Cents
	Description string    `json:"description"`
	CategoryID  int64     `json:"category_id"`
	Date        time.Time `json:"date"`
}

// ListFilter defines filtering and pagination for listing expenses.
type ListFilter struct {
	From     time.Time // Inclusive start date
	To       time.Time // Exclusive end date
	Category *int64    // Optional category ID filter
	Limit    int
	Offset   int
}

// Summary holds aggregated expense data for a time period.
type Summary struct {
	Total        int64           `json:"total"` // Total in cents
	ByCategory   []CategoryTotal `json:"by_category"`
	ExpenseCount int             `json:"expense_count"`
}

// CategoryTotal is a category with its total amount.
type CategoryTotal struct {
	CategoryID   int64  `json:"category_id"`
	CategoryName string `json:"category_name"`
	Total        int64  `json:"total"`
}

// CategoryChecker validates that a category exists.
// Defined here so expense package doesn't import category package.
// category.Service satisfies this interface implicitly.
type CategoryChecker interface {
	Exists(ctx context.Context, id int64) (bool, error)
}
