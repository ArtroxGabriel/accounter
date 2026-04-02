package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ArtroxGabriel/accounter/internal/config"
	"github.com/ArtroxGabriel/accounter/internal/platform/migrate"
	"github.com/samber/do/v2"
	_ "modernc.org/sqlite" // SQLite driver
)

type Database struct {
	db *sql.DB
}

var _ do.ShutdownerWithContextAndError = (*Database)(nil)

func New(i do.Injector) (*Database, error) {
	c := do.MustInvoke[config.Config](i)
	ctx := context.Background()
	db, openErr := Open(ctx, c.DatabasePath)
	if openErr != nil {
		return nil, openErr
	}

	if migrateErr := migrate.Run(ctx, db); migrateErr != nil {
		return nil, migrateErr
	}

	return &Database{db: db}, nil
}

// NewFromDB wraps an existing [sql.DB], primarily for tests to mock the database container.
func NewFromDB(db *sql.DB) *Database {
	return &Database{db: db}
}

func (d *Database) DB() *sql.DB {
	return d.db
}

func (d *Database) Shutdown(context.Context) error {
	if closeErr := d.db.Close(); closeErr != nil {
		return fmt.Errorf("database close failed: %w", closeErr)
	}
	return nil
}

// Open opens a SQLite connection and applies necessary PRAGMAs.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	// For modernc.org/sqlite, the driver name is "sqlite"
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite: %w", err)
	}

	// PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;
	// WAL (Write-Ahead Logging) is better for concurrency.
	// Foreign keys are disabled by default in SQLite.
	if _, err = db.ExecContext(ctx, "PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("applying sqlite pragmas: %w", err)
	}

	return db, nil
}
