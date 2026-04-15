package dashboard

import (
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/samber/do/v2"

	"github.com/ArtroxGabriel/accounter/internal/category"
	"github.com/ArtroxGabriel/accounter/internal/config"
	"github.com/ArtroxGabriel/accounter/internal/expense"
	"github.com/ArtroxGabriel/accounter/web"
)

const (
	mb        = 1 << 20
	centsMult = 100
)

// Handler manages the web dashboard UI.
type Handler struct {
	expenseSvc  expense.Service
	categorySvc category.Service
	templates   *template.Template
	logger      *slog.Logger
	timezone    *time.Location
}

// NewHandler creates a dashboard handler using DI.
func NewHandler(i do.Injector) (*Handler, error) {
	cfg := do.MustInvoke[config.Config](i)
	expenseSvc := do.MustInvoke[expense.Service](i)
	categorySvc := do.MustInvoke[category.Service](i)
	logger := do.MustInvoke[*slog.Logger](i)

	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil, fmt.Errorf("loading timezone %s: %w", cfg.Timezone, err)
	}

	tmpl, err := web.LoadTemplates(web.TemplatesFS)
	if err != nil {
		return nil, fmt.Errorf("loading dashboard templates: %w", err)
	}

	return &Handler{
		expenseSvc:  expenseSvc,
		categorySvc: categorySvc,
		templates:   tmpl,
		logger:      logger,
		timezone:    loc,
	}, nil
}

// Routes registers dashboard routes.
func (h *Handler) Routes(r chi.Router) {
	r.Get("/", h.Index)
	r.Get("/expenses", h.ExpenseList)
	r.Post("/expenses", h.CreateExpense)
	r.Delete("/expenses/{id}", h.DeleteExpense)
	r.Get("/categories", h.CategoryList)
	r.Get("/categories/add", h.AddCategoryForm)
	r.Post("/categories", h.CreateCategory)
	r.Get("/summary", h.Summary)
	r.Get("/expense-summary", h.ExpenseSummary)
}

// Index renders the full dashboard page.
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := ParseFilterParams(r.URL.Query())
	listFilter, err := BuildListFilter(time.Now(), h.timezone, params)
	if err != nil {
		h.respondBadRequest(w, "invalid filters")
		return
	}

	cats, err := h.categorySvc.List(ctx)
	if err != nil {
		h.respondInternalError(w, r, "listing categories", err)
		return
	}

	expenses, err := h.expenseSvc.List(ctx, listFilter)
	if err != nil {
		h.respondInternalError(w, r, "listing expenses", err)
		return
	}

	summary, err := h.expenseSvc.Summary(ctx, listFilter.From, listFilter.To)
	if err != nil {
		h.respondInternalError(w, r, "loading summary", err)
		return
	}

	data := Data{
		Title:      "Overview",
		Categories: toCategoryViewModels(cats),
		Expenses:   h.toViewModels(expenses),
		Summary: SummaryViewModel{
			Total:        FormatCurrency(summary.Total),
			ExpenseCount: summary.ExpenseCount,
			Categories:   cats,
		},
		FilterParams: params,
	}

	h.render(w, r, "layout.html", data) // Layout for initial load
}

// ExpenseList returns the expense table partial.
func (h *Handler) ExpenseList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := ParseFilterParams(r.URL.Query())
	filter, err := BuildListFilter(time.Now(), h.timezone, params)
	if err != nil {
		h.respondBadRequest(w, "invalid filters")
		return
	}

	expenses, err := h.expenseSvc.List(ctx, filter)
	if err != nil {
		h.respondInternalError(w, r, "listing expenses", err)
		return
	}

	h.render(w, r, "expense-list", struct {
		Expenses []ExpenseViewModel
	}{
		Expenses: h.toViewModels(expenses),
	})
}

// CreateExpense handles form submission and returns a new row.
func (h *Handler) CreateExpense(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	r.Body = http.MaxBytesReader(w, r.Body, mb) // 1MB limit for form bodies
	if err := r.ParseForm(); err != nil {
		h.respondBadRequest(w, "invalid form payload")
		return
	}

	amount, err := strconv.ParseFloat(r.FormValue("amount"), 64)
	if err != nil {
		h.respondBadRequest(w, "invalid amount")
		return
	}

	catID, err := strconv.ParseInt(r.FormValue("category_id"), 10, 64)
	if err != nil {
		h.respondBadRequest(w, "invalid category")
		return
	}

	desc := r.FormValue("description")

	dateStr := r.FormValue("date")
	date := time.Now().In(h.timezone)
	if dateStr != "" {
		parsedDate, parseErr := time.ParseInLocation(dateOnly, dateStr, h.timezone)
		if parseErr != nil {
			h.respondBadRequest(w, "invalid date")
			return
		}

		date = parsedDate
	}

	exp, err := h.expenseSvc.Create(ctx, expense.CreateExpenseInput{
		Amount:      int64(amount * centsMult), // cents conversion
		Description: desc,
		CategoryID:  catID,
		Date:        date,
	})
	if err != nil {
		h.respondBadRequest(w, "failed to create expense")
		return
	}

	// Trigger related updates in UI
	w.Header().Add("Hx-Trigger", "expense-updated")

	h.render(w, r, "expense-row", ToExpenseViewModel(exp, h.timezone))
}

// DeleteExpense removes an expense.
func (h *Handler) DeleteExpense(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.respondBadRequest(w, "invalid expense id")
		return
	}

	if err := h.expenseSvc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, expense.ErrNotFound) {
			h.respondNotFound(w, "expense not found")
			return
		}
		h.respondInternalError(w, r, "deleting expense", err)
		return
	}
	w.Header().Add("Hx-Trigger", "expense-updated")
	w.WriteHeader(http.StatusNoContent)
}

// CategoryList returns the category list fragments.
func (h *Handler) CategoryList(w http.ResponseWriter, r *http.Request) {
	cats, err := h.categorySvc.List(r.Context())
	if err != nil {
		h.respondInternalError(w, r, "listing categories", err)
		return
	}

	h.render(w, r, "category-list", struct {
		Categories []CategoryViewModel
	}{
		Categories: toCategoryViewModels(cats),
	})
}

// AddCategoryForm returns the form fragment for adding categories.
func (h *Handler) AddCategoryForm(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "category-form", nil)
}

// CreateCategory handles category creation.
func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, mb) // 1MB limit for form bodies
	if err := r.ParseForm(); err != nil {
		h.respondBadRequest(w, "invalid form payload")
		return
	}

	name := r.FormValue("name")
	icon := r.FormValue("icon")

	cat, err := h.categorySvc.Create(r.Context(), category.CreateCategoryInput{
		Name: name,
		Icon: icon,
	})
	if err != nil {
		h.respondBadRequest(w, "failed to create category")
		return
	}

	h.render(w, r, "category-list", struct {
		Categories []CategoryViewModel
	}{
		Categories: []CategoryViewModel{{ID: cat.ID, Name: cat.Name, Icon: cat.Icon}},
	})
}

// Summary returns the hero total balance.
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := ParseFilterParams(r.URL.Query())
	filter, err := BuildListFilter(time.Now(), h.timezone, params)
	if err != nil {
		h.respondBadRequest(w, "invalid filters")
		return
	}

	summary, err := h.expenseSvc.Summary(ctx, filter.From, filter.To)
	if err != nil {
		h.respondInternalError(w, r, "loading summary", err)
		return
	}

	categories, err := h.categorySvc.List(ctx)
	if err != nil {
		h.respondInternalError(w, r, "listing categories", err)
		return
	}

	h.render(w, r, "summary", struct {
		Total        string
		ExpenseCount int
		Categories   []category.Category
	}{
		Total:        FormatCurrency(summary.Total),
		ExpenseCount: summary.ExpenseCount,
		Categories:   categories,
	})
}

// ExpenseSummary returns the compact expense summary bar.
func (h *Handler) ExpenseSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := ParseFilterParams(r.URL.Query())
	filter, err := BuildListFilter(time.Now(), h.timezone, params)
	if err != nil {
		h.respondBadRequest(w, "invalid filters")
		return
	}

	summary, err := h.expenseSvc.Summary(ctx, filter.From, filter.To)
	if err != nil {
		h.respondInternalError(w, r, "loading summary", err)
		return
	}

	h.render(w, r, "expense-summary-bar", struct {
		Total        string
		ExpenseCount int
	}{
		Total:        FormatCurrency(summary.Total),
		ExpenseCount: summary.ExpenseCount,
	})
}

func toCategoryViewModels(categories []category.Category) []CategoryViewModel {
	result := make([]CategoryViewModel, 0, len(categories))
	for _, cat := range categories {
		result = append(result, CategoryViewModel{ID: cat.ID, Name: cat.Name, Icon: cat.Icon})
	}

	return result
}

func (h *Handler) toViewModels(expenses []expense.Expense) []ExpenseViewModel {
	vms := make([]ExpenseViewModel, len(expenses))
	for i, e := range expenses {
		vms[i] = ToExpenseViewModel(e, h.timezone)
	}
	return vms
}

func (h *Handler) render(w http.ResponseWriter, r *http.Request, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if r.Header.Get("Hx-Request") == "" && name != "layout.html" {
		// If not HTMX, wrap in layout (simplistic)
		name = "layout.html"
	}

	if err := h.executeTemplate(w, name, data); err != nil {
		h.logger.ErrorContext(r.Context(), "template failure", "template", name, "error", err)
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func (h *Handler) executeTemplate(w http.ResponseWriter, name string, data any) error {
	templateCandidates := []string{
		name,
		filepath.Join("templates", name),
		filepath.Base(name),
	}

	for _, candidate := range templateCandidates {
		if h.templates.Lookup(candidate) == nil {
			continue
		}

		if err := h.templates.ExecuteTemplate(w, candidate, data); err != nil {
			return fmt.Errorf("executing template %q: %w", candidate, err)
		}

		return nil
	}

	return fmt.Errorf("template %q not found", name)
}

func (h *Handler) respondBadRequest(w http.ResponseWriter, message string) {
	http.Error(w, message, http.StatusBadRequest)
}

func (h *Handler) respondNotFound(w http.ResponseWriter, message string) {
	http.Error(w, message, http.StatusNotFound)
}

func (h *Handler) respondInternalError(w http.ResponseWriter, r *http.Request, operation string, err error) {
	h.logger.ErrorContext(r.Context(), operation+" failed", "error", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}
