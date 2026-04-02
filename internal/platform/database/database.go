package database

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // SQLite driver
)

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
