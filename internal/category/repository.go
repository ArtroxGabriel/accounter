package category

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("not found")

// Repository provides access to the category storage.
type Repository interface {
	Create(ctx context.Context, input CreateCategoryInput) (Category, error)
	GetByID(ctx context.Context, id int64) (Category, error)
	GetByName(ctx context.Context, name string) (Category, error)
	List(ctx context.Context) ([]Category, error)
	Exists(ctx context.Context, id int64) (bool, error)
	Update(ctx context.Context, cat Category) (Category, error)
	Delete(ctx context.Context, id int64) error
}
