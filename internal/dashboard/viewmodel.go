package dashboard

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ArtroxGabriel/accounter/internal/category"
	"github.com/ArtroxGabriel/accounter/internal/expense"
)

const divisor = 100.0

const (
	periodToday = "today"
	periodWeek  = "week"
	periodMonth = "month"
	periodYear  = "year"
	periodAll   = "all"
	dateOnly    = "2006-01-02"
	oneWeek     = 7
)

const allTimeStartYear = 1970

var (
	ErrInvalidPeriod     = errors.New("invalid period")
	ErrInvalidDateFilter = errors.New("invalid date filter")
)

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

// ParseFilterParams extracts dashboard filters from query values.
func ParseFilterParams(values url.Values) FilterParams {
	period := strings.TrimSpace(values.Get("period"))
	if period == "" {
		period = periodMonth
	}

	return FilterParams{
		Period: period,
		From:   strings.TrimSpace(values.Get("from")),
		To:     strings.TrimSpace(values.Get("to")),
	}
}

// BuildListFilter computes date ranges for expense list queries.
func BuildListFilter(now time.Time, tz *time.Location, params FilterParams) (expense.ListFilter, error) {
	localizedNow := now.In(tz)

	if params.From != "" || params.To != "" {
		return buildCustomPeriodo(localizedNow, params.From, params.To, tz)
	}

	switch params.Period {
	case periodToday:
		from := time.Date(localizedNow.Year(), localizedNow.Month(), localizedNow.Day(), 0, 0, 0, 0, tz)
		return expense.ListFilter{From: from, To: from.AddDate(0, 0, 1)}, nil
	case periodWeek:
		weekdayOffset := int(localizedNow.Weekday())
		if weekdayOffset == 0 {
			weekdayOffset = 7
		}
		from := time.Date(
			localizedNow.Year(),
			localizedNow.Month(),
			localizedNow.Day()-weekdayOffset+1,
			0,
			0,
			0,
			0,
			tz,
		)
		return expense.ListFilter{From: from, To: from.AddDate(0, 0, oneWeek)}, nil
	case periodMonth:
		from := time.Date(localizedNow.Year(), localizedNow.Month(), 1, 0, 0, 0, 0, tz)
		return expense.ListFilter{From: from, To: from.AddDate(0, 1, 0)}, nil
	case periodYear:
		from := time.Date(localizedNow.Year(), 1, 1, 0, 0, 0, 0, tz)
		return expense.ListFilter{From: from, To: from.AddDate(1, 0, 0)}, nil
	case periodAll:
		from := time.Date(allTimeStartYear, 1, 1, 0, 0, 0, 0, tz)
		to := localizedNow.AddDate(0, 0, 1)
		return expense.ListFilter{From: from, To: to}, nil
	default:
		return expense.ListFilter{}, ErrInvalidPeriod
	}
}

func buildCustomPeriodo(
	localizedNow time.Time,
	startDate, endDate string,
	tz *time.Location,
) (expense.ListFilter, error) {
	filter := expense.ListFilter{}

	if startDate != "" {
		from, err := time.ParseInLocation(dateOnly, endDate, tz)
		if err != nil {
			return expense.ListFilter{}, ErrInvalidDateFilter
		}
		filter.From = from
	}

	if endDate != "" {
		toDate, err := time.ParseInLocation(dateOnly, endDate, tz)
		if err != nil {
			return expense.ListFilter{}, ErrInvalidDateFilter
		}
		filter.To = toDate.AddDate(0, 0, 1)
	}

	if filter.From.IsZero() {
		filter.From = time.Date(allTimeStartYear, 1, 1, 0, 0, 0, 0, tz)
	}

	if filter.To.IsZero() {
		filter.To = localizedNow.AddDate(0, 0, 1)
	}

	if !filter.From.IsZero() && !filter.To.IsZero() && !filter.From.Before(filter.To) {
		return expense.ListFilter{}, ErrInvalidDateFilter
	}

	return filter, nil
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
