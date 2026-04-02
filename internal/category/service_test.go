package category_test

import (
	"context"
	"testing"
	"time"

	"github.com/ArtroxGabriel/accounter/internal/category"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupTestService(t *testing.T, repo category.Repository) category.Service {
	t.Helper()
	injector := do.New()
	do.ProvideValue[category.Repository](injector, repo)
	svc, err := category.NewService(injector)
	require.NoError(t, err)
	return svc
}

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, input category.CreateCategoryInput) (category.Category, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(category.Category), args.Error(1)
}

func (m *MockRepository) GetByID(ctx context.Context, id int64) (category.Category, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(category.Category), args.Error(1)
}

func (m *MockRepository) GetByName(ctx context.Context, name string) (category.Category, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(category.Category), args.Error(1)
}

func (m *MockRepository) List(ctx context.Context) ([]category.Category, error) {
	args := m.Called(ctx)
	return args.Get(0).([]category.Category), args.Error(1)
}

func (m *MockRepository) Exists(ctx context.Context, id int64) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) Update(ctx context.Context, cat category.Category) (category.Category, error) {
	args := m.Called(ctx, cat)
	return args.Get(0).(category.Category), args.Error(1)
}

func (m *MockRepository) Delete(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestDefaultService_Create(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		input      category.CreateCategoryInput
		setupMock  func(*MockRepository)
		wantErr    error
		wantResult category.Category
	}{
		{
			name: "Create with valid input delegates to repo and returns result",
			input: category.CreateCategoryInput{
				Name: "Test Category",
				Icon: "🎯",
			},
			setupMock: func(m *MockRepository) {
				m.On("Create", ctx, category.CreateCategoryInput{
					Name: "Test Category",
					Icon: "🎯",
				}).Return(category.Category{ID: 1, Name: "Test Category", Icon: "🎯", CreatedAt: time.Now()}, nil)
			},
			wantResult: category.Category{ID: 1, Name: "Test Category", Icon: "🎯"},
			wantErr:    nil,
		},
		{
			name: "Create with empty name -> validation error",
			input: category.CreateCategoryInput{
				Name: "  ",
				Icon: "🎯",
			},
			setupMock:  func(_ *MockRepository) {},
			wantResult: category.Category{},
			wantErr:    category.ErrInvalidInput,
		},
		{
			name: "Create with empty icon defaults to package box",
			input: category.CreateCategoryInput{
				Name: "No Icon",
				Icon: "",
			},
			setupMock: func(m *MockRepository) {
				m.On("Create", ctx, category.CreateCategoryInput{
					Name: "No Icon",
					Icon: "📦",
				}).Return(category.Category{ID: 2, Name: "No Icon", Icon: "📦", CreatedAt: time.Now()}, nil)
			},
			wantResult: category.Category{ID: 2, Name: "No Icon", Icon: "📦"},
			wantErr:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockRepository)
			tt.setupMock(repo)
			svc := setupTestService(t, repo)

			got, err := svc.Create(ctx, tt.input)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantResult.ID, got.ID)
			assert.Equal(t, tt.wantResult.Name, got.Name)
			assert.Equal(t, tt.wantResult.Icon, got.Icon)
			repo.AssertExpectations(t)
		})
	}
}

func TestDefaultService_List(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	svc := setupTestService(t, repo)

	expected := []category.Category{{ID: 1, Name: "A"}}
	repo.On("List", ctx).Return(expected, nil)

	got, err := svc.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, expected, got)
	repo.AssertExpectations(t)
}

func TestDefaultService_GetByID(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	svc := setupTestService(t, repo)

	expected := category.Category{ID: 1, Name: "A"}
	repo.On("GetByID", ctx, int64(1)).Return(expected, nil)

	got, err := svc.GetByID(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, expected, got)
	repo.AssertExpectations(t)
}

func TestDefaultService_GetByName(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	svc := setupTestService(t, repo)

	expected := category.Category{ID: 1, Name: "A"}
	repo.On("GetByName", ctx, "A").Return(expected, nil)

	got, err := svc.GetByName(ctx, "A")
	require.NoError(t, err)
	assert.Equal(t, expected, got)
	repo.AssertExpectations(t)
}

func TestDefaultService_Exists(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	svc := setupTestService(t, repo)

	repo.On("Exists", ctx, int64(1)).Return(true, nil)

	got, err := svc.Exists(ctx, 1)
	require.NoError(t, err)
	assert.True(t, got)
	repo.AssertExpectations(t)
}

func TestDefaultService_Update(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	svc := setupTestService(t, repo)

	newName := "B"
	input := category.UpdateCategoryInput{Name: &newName}

	// Mock GetByID which is called internally by Update
	repo.On("GetByID", ctx, int64(1)).Return(category.Category{ID: 1, Name: "A", Icon: "X"}, nil)

	// Mock the actual Update with modified struct
	expected := category.Category{ID: 1, Name: "B", Icon: "X"}
	repo.On("Update", ctx, category.Category{ID: 1, Name: "B", Icon: "X"}).Return(expected, nil)

	got, err := svc.Update(ctx, 1, input)
	require.NoError(t, err)
	assert.Equal(t, expected, got)
	repo.AssertExpectations(t)
}

func TestDefaultService_Delete(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	svc := setupTestService(t, repo)

	repo.On("Delete", ctx, int64(1)).Return(nil)

	err := svc.Delete(ctx, 1)
	require.NoError(t, err)
	repo.AssertExpectations(t)
}
