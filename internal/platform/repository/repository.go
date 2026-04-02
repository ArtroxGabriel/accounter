package repository

import "context"

// Base defines the common CRUD operations for all domain repositories.
type Base[T any, CreateT any] interface {
	Create(ctx context.Context, input CreateT) (T, error)
	GetByID(ctx context.Context, id int64) (T, error)
	Delete(ctx context.Context, id int64) error
}
