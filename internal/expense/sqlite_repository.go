package expense

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ArtroxGabriel/accounter/internal/platform/database"
	"github.com/samber/do/v2"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(i do.Injector) (Repository, error) {
	dbContainer := do.MustInvoke[*database.Database](i)
	return &SQLiteRepository{db: dbContainer.DB()}, nil
}

func (r *SQLiteRepository) Create(ctx context.Context, e Expense) (Expense, error) {
	query := `
		INSERT INTO expenses (amount, description, category_id, date) 
		VALUES (?, ?, ?, ?) RETURNING id, amount, description, category_id, date, created_at
	`
	var exp Expense
	var dateStr, createdAtStr string

	dateToInsert := e.Date.Format(time.DateOnly)

	createErr := r.db.QueryRowContext(ctx, query, e.Amount, e.Description, e.CategoryID, dateToInsert).
		Scan(&exp.ID, &exp.Amount, &exp.Description, &exp.CategoryID, &dateStr, &createdAtStr)

	if createErr != nil {
		return Expense{}, createErr
	}

	// Fetch with joined category
	return r.GetByID(ctx, exp.ID)
}

func (r *SQLiteRepository) GetByID(ctx context.Context, id int64) (Expense, error) {
	query := `
		SELECT e.id, e.amount, e.description, e.category_id, c.name, e.date, e.created_at
		FROM expenses e
		JOIN categories c ON e.category_id = c.id
		WHERE e.id = ?
	`
	var exp Expense
	var dateStr, createdAtStr string

	getErr := r.db.QueryRowContext(ctx, query, id).
		Scan(&exp.ID, &exp.Amount, &exp.Description, &exp.CategoryID, &exp.Category, &dateStr, &createdAtStr)

	if errors.Is(getErr, sql.ErrNoRows) {
		return Expense{}, ErrNotFound
	}
	if getErr != nil {
		return Expense{}, getErr
	}

	exp.Date, _ = time.Parse(time.DateOnly, dateStr)
	exp.CreatedAt, _ = time.Parse(time.DateTime, createdAtStr)

	return exp, nil
}

func (r *SQLiteRepository) List(ctx context.Context, filter ListFilter) ([]Expense, error) {
	query := `
		SELECT e.id, e.amount, e.description, e.category_id, c.name, e.date, e.created_at
		FROM expenses e
		JOIN categories c ON e.category_id = c.id
		WHERE 1=1
	`
	var args []any

	if !filter.From.IsZero() {
		query += ` AND e.date >= ?`
		args = append(args, filter.From.Format(time.DateOnly))
	}
	if !filter.To.IsZero() {
		query += ` AND e.date < ?`
		args = append(args, filter.To.Format(time.DateOnly))
	}
	if filter.Category != nil {
		query += ` AND e.category_id = ?`
		args = append(args, *filter.Category)
	}

	query += ` ORDER BY e.date DESC, e.id DESC`

	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)

		if filter.Offset > 0 {
			query += ` OFFSET ?`
			args = append(args, filter.Offset)
		}
	}

	rows, queryErr := r.db.QueryContext(ctx, query, args...)
	if queryErr != nil {
		return nil, fmt.Errorf("listing expenses: %w", queryErr)
	}
	defer rows.Close()

	var expenses []Expense
	for rows.Next() {
		var exp Expense
		var dateStr, createdAtStr string
		if scanErr := rows.Scan(&exp.ID, &exp.Amount, &exp.Description,
			&exp.CategoryID, &exp.Category, &dateStr, &createdAtStr); scanErr != nil {
			return nil, scanErr
		}
		exp.Date, _ = time.Parse(time.DateOnly, strings.Split(dateStr, " ")[0]) // SQLite sometimes appends time
		exp.CreatedAt, _ = time.Parse(time.DateTime, createdAtStr)
		expenses = append(expenses, exp)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, rowsErr
	}

	return expenses, nil
}

func (r *SQLiteRepository) Delete(ctx context.Context, id int64) error {
	res, execErr := r.db.ExecContext(ctx, `DELETE FROM expenses WHERE id = ?`, id)
	if execErr != nil {
		return execErr
	}

	affected, affectedErr := res.RowsAffected()
	if affectedErr != nil {
		return affectedErr
	}

	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *SQLiteRepository) Summary(ctx context.Context, from, to time.Time) (Summary, error) {
	var s Summary

	// Query total and count
	totalQuery := `
		SELECT COALESCE(SUM(amount), 0), COUNT(id)
		FROM expenses
		WHERE date >= ? AND date < ?
	`
	fromStr := from.Format(time.DateOnly)
	toStr := to.Format(time.DateOnly)

	statsErr := r.db.QueryRowContext(ctx, totalQuery, fromStr, toStr).Scan(&s.Total, &s.ExpenseCount)
	if statsErr != nil {
		return s, statsErr
	}

	// Query breakdown
	catQuery := `
		SELECT c.id, c.name, SUM(e.amount) as total
		FROM expenses e
		JOIN categories c ON e.category_id = c.id
		WHERE e.date >= ? AND e.date < ?
		GROUP BY c.id, c.name
		ORDER BY total DESC
	`
	rows, rowsErr := r.db.QueryContext(ctx, catQuery, fromStr, toStr)
	if rowsErr != nil {
		return s, rowsErr
	}
	defer rows.Close()

	for rows.Next() {
		var ct CategoryTotal
		if scanErr := rows.Scan(&ct.CategoryID, &ct.CategoryName, &ct.Total); scanErr != nil {
			return s, scanErr
		}
		s.ByCategory = append(s.ByCategory, ct)
	}

	if err := rows.Err(); err != nil {
		return s, err
	}

	// Make ByCategory a non-nil empty slice when 0 elements, for better JSON
	if s.ByCategory == nil {
		s.ByCategory = make([]CategoryTotal, 0)
	}

	return s, nil
}
