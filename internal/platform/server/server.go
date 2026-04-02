package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/samber/do/v2"

	"github.com/ArtroxGabriel/accounter/internal/category"
	"github.com/ArtroxGabriel/accounter/internal/config"
	"github.com/ArtroxGabriel/accounter/internal/dashboard"
	"github.com/ArtroxGabriel/accounter/internal/expense"
	"github.com/ArtroxGabriel/accounter/internal/platform/auth"
)

const (
	defaultReadTimeout  = 5 * time.Second
	defaultWriteTimeout = 10 * time.Second
	defaultIdleTimeout  = 120 * time.Second
)

// Server represents the HTTP server.
type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

// New creates a new HTTP server.
func New(i do.Injector) (*Server, error) {
	cfg := do.MustInvoke[config.Config](i)
	logger := do.MustInvoke[*slog.Logger](i)

	categoryHandler := do.MustInvoke[*category.Handler](i)
	expenseHandler := do.MustInvoke[*expense.Handler](i)
	dashboardHandler := do.MustInvoke[*dashboard.Handler](i)

	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger) // Default for now, we can customize later

	// Public
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok"}`)
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(auth.BearerMiddleware(cfg.BearerToken))

		// JSON API
		r.Route("/api", func(r chi.Router) {
			r.Route("/categories", categoryHandler.Routes)
			r.Route("/expenses", expenseHandler.Routes)
		})

		// Web UI (HTMX)
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/dashboard", http.StatusFound)
		})
		r.Route("/dashboard", dashboardHandler.Routes)
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		IdleTimeout:  defaultIdleTimeout,
	}

	return &Server{
		httpServer: srv,
		logger:     logger.With(slog.String("port", cfg.Port)),
	}, nil
}

// Start runs the HTTP server.
func (s *Server) Start() error {
	s.logger.Info("Starting HTTP server")
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("starting server: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the server.
var _ do.ShutdownerWithContextAndError = (*Server)(nil)

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.InfoContext(ctx, "Shutting down HTTP server")
	return s.httpServer.Shutdown(ctx)
}
