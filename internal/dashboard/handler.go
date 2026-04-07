package dashboard

import (
	"fmt"
	"html/template"
	"io/fs"
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
	hoursDay  = 24
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

	var templateFiles []string
	if walkErr := fs.WalkDir(web.TemplatesFS, "templates", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() && filepath.Ext(path) == ".html" {
			templateFiles = append(templateFiles, path)
		}
		return nil
	}); walkErr != nil {
		return nil, fmt.Errorf("walking templates directory: %w", walkErr)
	}

	tmpl, err := template.ParseFS(web.TemplatesFS, templateFiles...)
	if err != nil {
		return nil, fmt.Errorf("parsing dashboard templates: %w", err)
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
	cats, _ := h.categorySvc.List(ctx)

	// Default to current month range
	now := time.Now().In(h.timezone)
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, h.timezone)
	to := from.AddDate(0, 1, 0)

	expenses, _ := h.expenseSvc.List(ctx, expense.ListFilter{From: from, To: to})

	data := struct {
		Title      string
		Categories []category.Category
		Expenses   []ExpenseViewModel
	}{
		Title:      "Overview",
		Categories: cats,
		Expenses:   h.toViewModels(expenses),
	}

	h.render(w, r, "layout.html", data) // Layout for initial load
}

// ExpenseList returns the expense table partial.
func (h *Handler) ExpenseList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Filters would be parsed from URL here...
	expenses, _ := h.expenseSvc.List(ctx, expense.ListFilter{}) // Simplified

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
		http.Error(w, "form too large", http.StatusBadRequest)
		return
	}

	amount, _ := strconv.ParseFloat(r.FormValue("amount"), 64)
	catID, _ := strconv.ParseInt(r.FormValue("category_id"), 10, 64)
	desc := r.FormValue("description")

	dateStr := r.FormValue("date")
	date := time.Now()
	if dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			date = t
		}
	}

	exp, err := h.expenseSvc.Create(ctx, expense.CreateExpenseInput{
		Amount:      int64(amount * centsMult), // cents conversion
		Description: desc,
		CategoryID:  catID,
		Date:        date,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Trigger related updates in UI
	w.Header().Add("Hx-Trigger", "expense-updated")

	h.render(w, r, "expense-row", ToExpenseViewModel(exp, h.timezone))
}

// DeleteExpense removes an expense.
func (h *Handler) DeleteExpense(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.expenseSvc.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Add("Hx-Trigger", "expense-updated")
}

// CategoryList returns the category list fragments.
func (h *Handler) CategoryList(w http.ResponseWriter, r *http.Request) {
	cats, _ := h.categorySvc.List(r.Context())
	h.render(w, r, "category-list", struct {
		Categories []category.Category
	}{
		Categories: cats,
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
		http.Error(w, "form too large", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	icon := r.FormValue("icon")

	cat, err := h.categorySvc.Create(r.Context(), category.CreateCategoryInput{
		Name: name,
		Icon: icon,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.render(w, r, "category-list", struct {
		Categories []category.Category
	}{
		Categories: []category.Category{cat},
	})
}

// Summary returns the hero total balance.
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	summary, _ := h.expenseSvc.Summary(ctx, time.Now().AddDate(0, -1, 0), time.Now().Add(hoursDay*time.Hour))

	h.render(w, r, "summary", struct {
		Total string
	}{
		Total: FormatCurrency(summary.Total),
	})
}

// ExpenseSummary returns the compact expense summary bar.
func (h *Handler) ExpenseSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	summary, _ := h.expenseSvc.Summary(ctx, time.Now().AddDate(0, -1, 0), time.Now().Add(hoursDay*time.Hour))

	h.render(w, r, "expense-summary-bar", struct {
		Total        string
		ExpenseCount int
	}{
		Total:        FormatCurrency(summary.Total),
		ExpenseCount: summary.ExpenseCount,
	})
}

func (h *Handler) toViewModels(expenses []expense.Expense) []ExpenseViewModel {
	vms := make([]ExpenseViewModel, len(expenses))
	for i, e := range expenses {
		vms[i] = ToExpenseViewModel(e, h.timezone)
	}
	return vms
}

func (h *Handler) render(w http.ResponseWriter, r *http.Request, name string, data any) {
	if r.Header.Get("Hx-Request") == "" && name != "layout.html" {
		// If not HTMX, wrap in layout (simplistic)
		name = "layout.html"
	}

	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		h.logger.Error("template failure", "template", name, "error", err)
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}
