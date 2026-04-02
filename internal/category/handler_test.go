package category_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ArtroxGabriel/accounter/internal/category"
	"github.com/go-chi/chi/v5"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) Create(ctx context.Context, input category.CreateCategoryInput) (category.Category, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(category.Category), args.Error(1)
}

func (m *MockService) GetByID(ctx context.Context, id int64) (category.Category, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(category.Category), args.Error(1)
}

func (m *MockService) GetByName(ctx context.Context, name string) (category.Category, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(category.Category), args.Error(1)
}

func (m *MockService) List(ctx context.Context) ([]category.Category, error) {
	args := m.Called(ctx)
	return args.Get(0).([]category.Category), args.Error(1)
}

func (m *MockService) Exists(ctx context.Context, id int64) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

func (m *MockService) Update(
	ctx context.Context,
	id int64,
	input category.UpdateCategoryInput,
) (category.Category, error) {
	args := m.Called(ctx, id, input)
	return args.Get(0).(category.Category), args.Error(1)
}

func (m *MockService) Delete(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func setupTestHandler(t *testing.T, s category.Service) *category.Handler {
	t.Helper()
	injector := do.New()
	do.ProvideValue[category.Service](injector, s)
	h, err := category.NewHandler(injector)
	require.NoError(t, err)
	return h
}

func TestHandler_Create(t *testing.T) {
	t.Run("valid input returns 201", func(t *testing.T) {
		svc := new(MockService)
		h := setupTestHandler(t, svc)

		input := category.CreateCategoryInput{Name: "Food", Icon: "🍔"}
		expected := category.Category{ID: 1, Name: "Food", Icon: "🍔", CreatedAt: time.Now()}

		svc.On("Create", mock.Anything, input).Return(expected, nil)

		body, _ := json.Marshal(input)
		req, _ := http.NewRequest(http.MethodPost, "/api/categories", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		h.Create(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		var actual category.Category
		json.Unmarshal(rr.Body.Bytes(), &actual)
		assert.Equal(t, expected.ID, actual.ID)
		assert.Equal(t, expected.Name, actual.Name)
		svc.AssertExpectations(t)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		svc := new(MockService)
		h := setupTestHandler(t, svc)

		req, _ := http.NewRequest(http.MethodPost, "/api/categories", bytes.NewBufferString("invalid json"))
		rr := httptest.NewRecorder()

		h.Create(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("service error returns 400", func(t *testing.T) {
		svc := new(MockService)
		h := setupTestHandler(t, svc)

		input := category.CreateCategoryInput{Name: "", Icon: "🍔"}
		svc.On("Create", mock.Anything, input).Return(category.Category{}, category.ErrInvalidInput)

		body, _ := json.Marshal(input)
		req, _ := http.NewRequest(http.MethodPost, "/api/categories", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		h.Create(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestHandler_List(t *testing.T) {
	svc := new(MockService)
	h := setupTestHandler(t, svc)

	expected := []category.Category{{ID: 1, Name: "Food"}}
	svc.On("List", mock.Anything).Return(expected, nil)

	req, _ := http.NewRequest(http.MethodGet, "/api/categories", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var actual []category.Category
	json.Unmarshal(rr.Body.Bytes(), &actual)
	assert.Len(t, actual, 1)
	assert.Equal(t, expected[0].Name, actual[0].Name)
}

func TestHandler_Update(t *testing.T) {
	t.Run("valid update returns 200", func(t *testing.T) {
		svc := new(MockService)
		h := setupTestHandler(t, svc)

		newName := "Groceries"
		input := category.UpdateCategoryInput{Name: &newName}
		expected := category.Category{ID: 1, Name: "Groceries", Icon: "🍎"}

		svc.On("Update", mock.Anything, int64(1), input).Return(expected, nil)

		body, _ := json.Marshal(input)
		req, _ := http.NewRequest(http.MethodPut, "/api/categories/1", bytes.NewBuffer(body))

		// Setup chi context
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "1")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()
		h.Update(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var actual category.Category
		json.Unmarshal(rr.Body.Bytes(), &actual)
		assert.Equal(t, expected.Name, actual.Name)
	})

	t.Run("invalid ID returns 400", func(t *testing.T) {
		svc := new(MockService)
		h := setupTestHandler(t, svc)

		req, _ := http.NewRequest(http.MethodPut, "/api/categories/abc", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "abc")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()
		h.Update(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestHandler_Delete(t *testing.T) {
	t.Run("successful delete returns 204", func(t *testing.T) {
		svc := new(MockService)
		h := setupTestHandler(t, svc)

		svc.On("Delete", mock.Anything, int64(1)).Return(nil)

		req, _ := http.NewRequest(http.MethodDelete, "/api/categories/1", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "1")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()
		h.Delete(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("delete failed (e.g. in use) returns 400", func(t *testing.T) {
		svc := new(MockService)
		h := setupTestHandler(t, svc)

		svc.On("Delete", mock.Anything, int64(1)).Return(errors.New("in use"))

		req, _ := http.NewRequest(http.MethodDelete, "/api/categories/1", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "1")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()
		h.Delete(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
