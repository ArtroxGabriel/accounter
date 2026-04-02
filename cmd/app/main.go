package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ArtroxGabriel/accounter/internal/category"
	"github.com/ArtroxGabriel/accounter/internal/config"
	"github.com/ArtroxGabriel/accounter/internal/platform/database"
	"github.com/ArtroxGabriel/accounter/internal/platform/logger"
	"github.com/ArtroxGabriel/accounter/internal/platform/migrate"
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
	do.Provide(injector, func(i do.Injector) (*slog.Logger, error) {
		c := do.MustInvoke[config.Config](i)
		return logger.New(c.LogLevel, c.Environment), nil
	})

	// 3. Database
	do.Provide(injector, func(i do.Injector) (*sql.DB, error) {
		c := do.MustInvoke[config.Config](i)
		ctx := context.Background()
		db, openErr := database.Open(ctx, c.DatabasePath)
		if openErr != nil {
			return nil, openErr
		}

		if migrateErr := migrate.Run(ctx, db); migrateErr != nil {
			return nil, migrateErr
		}

		return db, nil
	})

	// 4. Category Repository
	do.Provide(injector, category.NewSQLiteRepository)

	// 5. Category Service
	do.Provide(injector, category.NewService)

	l := do.MustInvoke[*slog.Logger](injector)
	l.Info("Accounter — Starting...", "env", cfg.Environment)

	// Verify instantiation
	_ = do.MustInvoke[category.Service](injector)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	<-ctx.Done()
	cancel() // Release context early since we caught the signal
	l.Info("Shutting down...")

	// Close database explicitly
	if db, invokeErr := do.Invoke[*sql.DB](injector); invokeErr == nil {
		if closeErr := db.Close(); closeErr != nil {
			l.Error("database close failed", "error", closeErr)
		}
	}

	// Shutdown services bound in do
	if shutdownErr := injector.Shutdown(); shutdownErr != nil {
		l.Error("shutdown failed", "error", shutdownErr)
		os.Exit(1)
	}

	l.Info("Shutdown complete.")
}
