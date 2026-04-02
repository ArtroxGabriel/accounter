package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ArtroxGabriel/accounter/internal/category"
	"github.com/ArtroxGabriel/accounter/internal/config"
	"github.com/ArtroxGabriel/accounter/internal/dashboard"
	"github.com/ArtroxGabriel/accounter/internal/expense"
	"github.com/ArtroxGabriel/accounter/internal/platform/database"
	"github.com/ArtroxGabriel/accounter/internal/platform/logger"
	"github.com/ArtroxGabriel/accounter/internal/platform/server"
	"github.com/samber/do/v2"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	injector := do.New()

	// 1. Config
	do.ProvideValue(injector, cfg)

	// 2. Logger
	do.ProvideValue(injector, logger.New(cfg.LogLevel, cfg.Environment))

	// 3. Database
	database.Package(injector)

	// 4. Domain Packages
	category.Package(injector)
	expense.Package(injector)
	dashboard.Package(injector)

	// 5. Server
	server.Package(injector)

	l := do.MustInvoke[*slog.Logger](injector)
	l.Info("Accounter — Starting...", "env", cfg.Environment)

	srv := do.MustInvoke[*server.Server](injector)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start server in background
	go func() {
		if startErr := srv.Start(); startErr != nil {
			l.Error("server error", "error", startErr)
			stop()
		}
	}()

	// Wait for termination signal
	<-ctx.Done()
	l.Info("Shutting down...")

	// Shutdown services bound in do
	if shutdownErr := injector.Shutdown(); shutdownErr != nil {
		l.Error("shutdown failed", "error", shutdownErr)
	}

	l.Info("Shutdown complete.")
}
