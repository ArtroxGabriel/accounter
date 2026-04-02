package expense

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/samber/do/v2"
)

// Handler handles HTTP requests for the expense domain.
type Handler struct {
	svc Service
}

// NewHandler creates a new expense handler using DI.
func NewHandler(i do.Injector) (*Handler, error) {
	svc := do.MustInvoke[Service](i)
	return &Handler{svc: svc}, nil
}

// Routes registers the expense routes to the provided router.
func (h *Handler) Routes(r chi.Router) {
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/summary", h.Summary)
	r.Get("/{id}", h.Get)
	r.Delete("/{id}", h.Delete)
}

const defaultLimit = 100

// Create handles the POST /api/expenses request.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var input CreateExpenseInput
	if decErr := json.NewDecoder(r.Body).Decode(&input); decErr != nil {
		h.error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	exp, err := h.svc.Create(r.Context(), input)
	if err != nil {
		h.error(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respond(w, http.StatusCreated, exp)
}

// List handles the GET /api/expenses request.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := ListFilter{
		Limit:  defaultLimit,
		Offset: 0,
	}

	if from := query.Get("from"); from != "" {
		if t, err := time.Parse("2006-01-02", from); err == nil {
			filter.From = t
		}
	}
	if to := query.Get("to"); to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			filter.To = t
		}
	}
	if catID := query.Get("category_id"); catID != "" {
		if id, err := strconv.ParseInt(catID, 10, 64); err == nil {
			filter.Category = &id
		}
	}
	if limit := query.Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filter.Limit = l
		}
	}
	if offset := query.Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			filter.Offset = o
		}
	}

	expenses, err := h.svc.List(r.Context(), filter)
	if err != nil {
		h.error(w, http.StatusInternalServerError, "failed to list expenses")
		return
	}

	h.respond(w, http.StatusOK, expenses)
}

// Get handles the GET /api/expenses/{id} request.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.error(w, http.StatusBadRequest, "invalid expense id")
		return
	}

	exp, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		h.error(w, http.StatusNotFound, "expense not found")
		return
	}

	h.respond(w, http.StatusOK, exp)
}

// Summary handles the GET /api/expenses/summary request.
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	fromStr := query.Get("from")
	toStr := query.Get("to")

	if fromStr == "" || toStr == "" {
		h.error(w, http.StatusBadRequest, "missing from or to query parameters")
		return
	}

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		h.error(w, http.StatusBadRequest, "invalid from date")
		return
	}

	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		h.error(w, http.StatusBadRequest, "invalid to date")
		return
	}

	summary, err := h.svc.Summary(r.Context(), from, to)
	if err != nil {
		h.error(w, http.StatusInternalServerError, "failed to get summary")
		return
	}

	h.respond(w, http.StatusOK, summary)
}

// Delete handles the DELETE /api/expenses/{id} request.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.error(w, http.StatusBadRequest, "invalid expense id")
		return
	}

	if delErr := h.svc.Delete(r.Context(), id); delErr != nil {
		h.error(w, http.StatusInternalServerError, "failed to delete expense")
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
