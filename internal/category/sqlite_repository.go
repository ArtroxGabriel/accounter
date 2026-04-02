package category

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/samber/do/v2"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(i do.Injector) (Repository, error) {
	db := do.MustInvoke[*sql.DB](i)
	return &SQLiteRepository{db: db}, nil
}

func (r *SQLiteRepository) Create(ctx context.Context, input CreateCategoryInput) (Category, error) {
	query := `INSERT INTO categories (name, icon) VALUES (?, ?) RETURNING id, name, icon, created_at`
	var cat Category
	var createdAt string
	createErr := r.db.QueryRowContext(ctx, query, input.Name, input.Icon).
		Scan(&cat.ID, &cat.Name, &cat.Icon, &createdAt)
	if createErr != nil {
		return Category{}, createErr
	}
	cat.CreatedAt, _ = time.Parse(time.DateTime, createdAt)
	return cat, nil
}

func (r *SQLiteRepository) GetByID(ctx context.Context, id int64) (Category, error) {
	var cat Category
	var createdAt string
	getErr := r.db.QueryRowContext(ctx, `SELECT id, name, icon, created_at FROM categories WHERE id = ?`, id).
		Scan(&cat.ID, &cat.Name, &cat.Icon, &createdAt)
	if errors.Is(getErr, sql.ErrNoRows) {
		return Category{}, ErrNotFound
	}
	if getErr != nil {
		return Category{}, getErr
	}
	cat.CreatedAt, _ = time.Parse(time.DateTime, createdAt)
	return cat, nil
}

func (r *SQLiteRepository) GetByName(ctx context.Context, name string) (Category, error) {
	var cat Category
	var createdAt string
	getErr := r.db.QueryRowContext(ctx, `SELECT id, name, icon, created_at FROM categories WHERE LOWER(name) = ?`, strings.ToLower(name)).
		Scan(&cat.ID, &cat.Name, &cat.Icon, &createdAt)
	if errors.Is(getErr, sql.ErrNoRows) {
		return Category{}, ErrNotFound
	}
	if getErr != nil {
		return Category{}, getErr
	}
	cat.CreatedAt, _ = time.Parse(time.DateTime, createdAt)
	return cat, nil
}

func (r *SQLiteRepository) List(ctx context.Context) ([]Category, error) {
	rows, queryErr := r.db.QueryContext(ctx, `SELECT id, name, icon, created_at FROM categories ORDER BY name ASC`)
	if queryErr != nil {
		return nil, queryErr
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var cat Category
		var createdAt string
		if scanErr := rows.Scan(&cat.ID, &cat.Name, &cat.Icon, &createdAt); scanErr != nil {
			return nil, scanErr
		}
		cat.CreatedAt, _ = time.Parse(time.DateTime, createdAt)
		categories = append(categories, cat)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, rowsErr
	}
	return categories, nil
}

func (r *SQLiteRepository) Exists(ctx context.Context, id int64) (bool, error) {
	var exists bool
	existsErr := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM categories WHERE id = ?)`, id).Scan(&exists)
	if existsErr != nil {
		return false, existsErr
	}
	return exists, nil
}

func (r *SQLiteRepository) Update(ctx context.Context, cat Category) (Category, error) {
	_, execErr := r.db.ExecContext(ctx,
		`UPDATE categories SET name = ?, icon = ? WHERE id = ?`,
		cat.Name, cat.Icon, cat.ID)
	if execErr != nil {
		return Category{}, execErr
	}

	return r.GetByID(ctx, cat.ID)
}

func (r *SQLiteRepository) Delete(ctx context.Context, id int64) error {
	res, execErr := r.db.ExecContext(ctx, `DELETE FROM categories WHERE id = ?`, id)
	if execErr != nil {
		return execErr
	}
	affected, rowsErr := res.RowsAffected()
	if rowsErr != nil {
		return rowsErr
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}
