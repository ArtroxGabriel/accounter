package category

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/samber/do/v2"
)

var ErrInvalidInput = errors.New("invalid input")

// Service provides business logic for categories.
type Service interface {
	Create(ctx context.Context, input CreateCategoryInput) (Category, error)
	GetByID(ctx context.Context, id int64) (Category, error)
	GetByName(ctx context.Context, name string) (Category, error)
	List(ctx context.Context) ([]Category, error)
	Exists(ctx context.Context, id int64) (bool, error)
	Update(ctx context.Context, id int64, input UpdateCategoryInput) (Category, error)
	Delete(ctx context.Context, id int64) error
}

type DefaultService struct {
	repo Repository
}

func NewService(i do.Injector) (Service, error) {
	repo := do.MustInvoke[Repository](i)
	return &DefaultService{repo: repo}, nil
}

func (s *DefaultService) Create(ctx context.Context, input CreateCategoryInput) (Category, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return Category{}, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}

	input.Icon = strings.TrimSpace(input.Icon)
	if input.Icon == "" {
		input.Icon = "📦" // default icon
	}

	return s.repo.Create(ctx, input)
}

func (s *DefaultService) GetByID(ctx context.Context, id int64) (Category, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *DefaultService) GetByName(ctx context.Context, name string) (Category, error) {
	return s.repo.GetByName(ctx, name)
}

func (s *DefaultService) List(ctx context.Context) ([]Category, error) {
	return s.repo.List(ctx)
}

func (s *DefaultService) Exists(ctx context.Context, id int64) (bool, error) {
	return s.repo.Exists(ctx, id)
}

func (s *DefaultService) Update(ctx context.Context, id int64, input UpdateCategoryInput) (Category, error) {
	if input.Name != nil {
		newName := strings.TrimSpace(*input.Name)
		if newName == "" {
			return Category{}, fmt.Errorf("%w: name cannot be empty", ErrInvalidInput)
		}
		input.Name = &newName
	}
	if input.Icon != nil {
		newIcon := strings.TrimSpace(*input.Icon)
		if newIcon == "" {
			return Category{}, fmt.Errorf("%w: icon cannot be empty", ErrInvalidInput)
		}
		input.Icon = &newIcon
	}

	// Fetch, modify and use repo.Update(cat)
	cat, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Category{}, err
	}

	if input.Name != nil {
		cat.Name = *input.Name
	}
	if input.Icon != nil {
		cat.Icon = *input.Icon
	}

	return s.repo.Update(ctx, cat)
}

func (s *DefaultService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
