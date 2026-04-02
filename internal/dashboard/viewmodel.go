package dashboard

import (
	"fmt"
	"time"

	"github.com/ArtroxGabriel/accounter/internal/category"
	"github.com/ArtroxGabriel/accounter/internal/expense"
)

const divisor = 100.0

// Data holds all data needed to render the main dashboard.
type Data struct {
	Title        string
	Expenses     []ExpenseViewModel
	Categories   []CategoryViewModel
	Summary      SummaryViewModel
	FilterParams FilterParams
}

// ExpenseViewModel is a presentation-ready expense.
type ExpenseViewModel struct {
	ID              int64
	Amount          int64
	FormattedAmount string
	Description     string
	CategoryID      int64
	CategoryName    string
	CategoryIcon    string
	Date            time.Time
	FormattedDate   string
}

// CategoryViewModel is a presentation-ready category.
type CategoryViewModel struct {
	ID   int64
	Name string
	Icon string
}

// SummaryViewModel is a presentation-ready summary.
type SummaryViewModel struct {
	Total        string
	ExpenseCount int
	Categories   []category.Category
}

// FilterParams holds the current view filters.
type FilterParams struct {
	Period string
	From   string
	To     string
}

// ToExpenseViewModel converts a domain expense to a view model.
func ToExpenseViewModel(e expense.Expense, tz *time.Location) ExpenseViewModel {
	date := e.Date.In(tz)
	return ExpenseViewModel{
		ID:              e.ID,
		Amount:          e.Amount,
		FormattedAmount: FormatCurrency(e.Amount),
		Description:     e.Description,
		CategoryID:      e.CategoryID,
		CategoryName:    e.Category, // Denormalized name from JOIN
		CategoryIcon:    "",         // Will be populated by handler if needed or use joined data
		Date:            date,
		FormattedDate:   date.Format("02/01"), // DD/MM
	}
}

// FormatCurrency converts cents to a BRL currency string (e.g., 2550 -> "R$ 25,50").
func FormatCurrency(cents int64) string {
	return fmt.Sprintf("R$ %.2f", float64(cents)/divisor)
}
