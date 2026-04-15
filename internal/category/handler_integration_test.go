//go:build integration

package category_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/ArtroxGabriel/accounter/internal/category"
	"github.com/ArtroxGabriel/accounter/internal/platform/database"
	"github.com/ArtroxGabriel/accounter/internal/platform/migrate"
	"github.com/go-chi/chi/v5"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCategoryIntegrationRouter(t *testing.T) (http.Handler, *sql.DB) {
	t.Helper()

	ctx := context.Background()
	db, err := database.Open(ctx, ":memory:")
	require.NoError(t, err)
	require.NoError(t, migrate.Run(ctx, db))
	t.Cleanup(func() { _ = db.Close() })

	injector := do.New()
	do.ProvideValue(injector, database.NewFromDB(db))

	repo, err := category.NewSQLiteRepository(injector)
	require.NoError(t, err)
	do.ProvideValue[category.Repository](injector, repo)

	svc, err := category.NewService(injector)
	require.NoError(t, err)
	do.ProvideValue[category.Service](injector, svc)

	h, err := category.NewHandler(injector)
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Route("/api/categories", h.Routes)

	return r, db
}

func TestCategoryHandlerIntegration_CRUD(t *testing.T) {
	router, _ := setupCategoryIntegrationRouter(t)

	createBody := category.CreateCategoryInput{Name: "Food", Icon: "🍔"}
	body, err := json.Marshal(createBody)
	require.NoError(t, err)

	createReq, err := http.NewRequest(http.MethodPost, "/api/categories", bytes.NewReader(body))
	require.NoError(t, err)
	createReq.Header.Set("Content-Type", "application/json")

	createRes := httptest.NewRecorder()
	router.ServeHTTP(createRes, createReq)

	require.Equal(t, http.StatusCreated, createRes.Code)

	var created category.Category
	require.NoError(t, json.Unmarshal(createRes.Body.Bytes(), &created))
	assert.NotZero(t, created.ID)
	assert.Equal(t, "Food", created.Name)
	assert.Equal(t, "🍔", created.Icon)

	listReq, err := http.NewRequest(http.MethodGet, "/api/categories", nil)
	require.NoError(t, err)

	listRes := httptest.NewRecorder()
	router.ServeHTTP(listRes, listReq)

	require.Equal(t, http.StatusOK, listRes.Code)

	var listed []category.Category
	require.NoError(t, json.Unmarshal(listRes.Body.Bytes(), &listed))
	require.NotEmpty(t, listed)

	foundCreated := false
	for _, cat := range listed {
		if cat.ID == created.ID {
			foundCreated = true
			break
		}
	}
	assert.True(t, foundCreated)

	newName := "Groceries"
	updateBody := category.UpdateCategoryInput{Name: &newName}
	body, err = json.Marshal(updateBody)
	require.NoError(t, err)

	updateReq, err := http.NewRequest(http.MethodPut, "/api/categories/"+toString(created.ID), bytes.NewReader(body))
	require.NoError(t, err)
	updateReq.Header.Set("Content-Type", "application/json")

	updateRes := httptest.NewRecorder()
	router.ServeHTTP(updateRes, updateReq)

	require.Equal(t, http.StatusOK, updateRes.Code)

	var updated category.Category
	require.NoError(t, json.Unmarshal(updateRes.Body.Bytes(), &updated))
	assert.Equal(t, "Groceries", updated.Name)

	deleteReq, err := http.NewRequest(http.MethodDelete, "/api/categories/"+toString(created.ID), nil)
	require.NoError(t, err)

	deleteRes := httptest.NewRecorder()
	router.ServeHTTP(deleteRes, deleteReq)

	require.Equal(t, http.StatusNoContent, deleteRes.Code)

	deleteAgainReq, err := http.NewRequest(http.MethodDelete, "/api/categories/"+toString(created.ID), nil)
	require.NoError(t, err)

	deleteAgainRes := httptest.NewRecorder()
	router.ServeHTTP(deleteAgainRes, deleteAgainReq)

	assert.Equal(t, http.StatusBadRequest, deleteAgainRes.Code)
}

func TestCategoryHandlerIntegration_InvalidJSON(t *testing.T) {
	router, _ := setupCategoryIntegrationRouter(t)

	req, err := http.NewRequest(http.MethodPost, "/api/categories", bytes.NewBufferString("{invalid"))
	require.NoError(t, err)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusBadRequest, res.Code)
}

func TestCategoryHandlerIntegration_DeleteCategoryInUse(t *testing.T) {
	router, db := setupCategoryIntegrationRouter(t)

	createBody := category.CreateCategoryInput{Name: "Bills", Icon: "💡"}
	body, err := json.Marshal(createBody)
	require.NoError(t, err)

	createReq, err := http.NewRequest(http.MethodPost, "/api/categories", bytes.NewReader(body))
	require.NoError(t, err)

	createRes := httptest.NewRecorder()
	router.ServeHTTP(createRes, createReq)
	require.Equal(t, http.StatusCreated, createRes.Code)

	var created category.Category
	require.NoError(t, json.Unmarshal(createRes.Body.Bytes(), &created))

	_, err = db.ExecContext(context.Background(),
		"INSERT INTO expenses (amount, description, category_id, date) VALUES (?, ?, ?, ?)",
		1200, "Electricity", created.ID, time.Now().UTC().Format(time.DateOnly),
	)
	require.NoError(t, err)

	deleteReq, err := http.NewRequest(http.MethodDelete, "/api/categories/"+toString(created.ID), nil)
	require.NoError(t, err)

	deleteRes := httptest.NewRecorder()
	router.ServeHTTP(deleteRes, deleteReq)

	assert.Equal(t, http.StatusBadRequest, deleteRes.Code)
}

func toString(v int64) string {
	return strconv.FormatInt(v, 10)
}
