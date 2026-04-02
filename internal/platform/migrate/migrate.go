package migrate

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Run applies all pending migrations in order.
func Run(ctx context.Context, db *sql.DB) error {
	// 1. Create schema_migrations table if not exists
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    TEXT    NOT NULL UNIQUE,
		applied_at TEXT    NOT NULL DEFAULT (datetime('now'))
	);`)
	if err != nil {
		return fmt.Errorf("creating schema_migrations: %w", err)
	}

	// 2. Read all .sql files from embedded FS
	files, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("reading migrations directory: %w", err)
	}

	var filenames []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".sql") {
			filenames = append(filenames, f.Name())
		}
	}

	// 3. Sort by filename (lexicographic = version order)
	sort.Strings(filenames)

	// 4. For each migration not in schema_migrations
	for _, filename := range filenames {
		var exists bool
		err = db.
			QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)", filename).
			Scan(&exists)
		if err != nil {
			return fmt.Errorf("checking migration existence for %s: %w", filename, err)
		}

		if exists {
			continue
		}

		// Apply migration
		if err = applyMigration(ctx, db, filename); err != nil {
			return fmt.Errorf("applying migration %s: %w", filename, err)
		}
	}

	return nil
}

func applyMigration(ctx context.Context, db *sql.DB, filename string) error {
	content, err := migrationsFS.ReadFile("migrations/" + filename)
	if err != nil {
		return fmt.Errorf("reading migration file: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	// Rollback if something fails
	defer func() {
		_ = tx.Rollback()
	}()

	// Execute SQL
	_, err = tx.ExecContext(ctx, string(content))
	if err != nil {
		return fmt.Errorf("executing SQL: %w", err)
	}

	// Insert version into schema_migrations
	_, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES (?)", filename)
	if err != nil {
		return fmt.Errorf("recording migration: %w", err)
	}

	return tx.Commit()
}
