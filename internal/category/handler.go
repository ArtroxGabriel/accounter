package category

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/samber/do/v2"
)

// Handler handles HTTP requests for the category domain.
type Handler struct {
	svc Service
}

// NewHandler creates a new category handler using DI.
func NewHandler(i do.Injector) (*Handler, error) {
	svc := do.MustInvoke[Service](i)
	return &Handler{svc: svc}, nil
}

// Routes registers the category routes to the provided router.
func (h *Handler) Routes(r chi.Router) {
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
}

// Create handles the POST /api/categories request.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var input CreateCategoryInput
	if decErr := json.NewDecoder(r.Body).Decode(&input); decErr != nil {
		h.error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cat, err := h.svc.Create(r.Context(), input)
	if err != nil {
		h.error(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respond(w, http.StatusCreated, cat)
}

// List handles the GET /api/categories request.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	cats, err := h.svc.List(r.Context())
	if err != nil {
		h.error(w, http.StatusInternalServerError, "failed to list categories")
		return
	}

	h.respond(w, http.StatusOK, cats)
}

// Update handles the PUT /api/categories/{id} request.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.error(w, http.StatusBadRequest, "invalid category id")
		return
	}

	var input UpdateCategoryInput
	if decErr := json.NewDecoder(r.Body).Decode(&input); decErr != nil {
		h.error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cat, err := h.svc.Update(r.Context(), id, input)
	if err != nil {
		h.error(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respond(w, http.StatusOK, cat)
}

// Delete handles the DELETE /api/categories/{id} request.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.error(w, http.StatusBadRequest, "invalid category id")
		return
	}

	if delErr := h.svc.Delete(r.Context(), id); delErr != nil {
		h.error(w, http.StatusBadRequest, delErr.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// respond sends a JSON response.
func (h *Handler) respond(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

// error sends a JSON error response.
func (h *Handler) error(w http.ResponseWriter, code int, message string) {
	h.respond(w, code, map[string]string{"error": message})
}
