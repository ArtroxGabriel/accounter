package category

import "time"

// Category represents an expense category.
type Category struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon"` // Emoji or icon identifier
	CreatedAt time.Time `json:"created_at"`
}

// CreateCategoryInput is the input for creating a new category.
type CreateCategoryInput struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

// UpdateCategoryInput is the input for updating an existing category.
type UpdateCategoryInput struct {
	Name *string `json:"name,omitempty"`
	Icon *string `json:"icon,omitempty"`
}
