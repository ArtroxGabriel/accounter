package category

import (
	"context"
	"errors"

	"github.com/ArtroxGabriel/accounter/internal/platform/repository"
)

var ErrNotFound = errors.New("not found")

// Repository provides access to the category storage.
type Repository interface {
	repository.Base[Category, Category]
	GetByName(ctx context.Context, name string) (Category, error)
	List(ctx context.Context) ([]Category, error)
	Exists(ctx context.Context, id int64) (bool, error)
	Update(ctx context.Context, cat Category) (Category, error)
}
