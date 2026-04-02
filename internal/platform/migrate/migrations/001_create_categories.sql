-- internal/platform/migrate/migrations/001_create_categories.sql

CREATE TABLE IF NOT EXISTS categories (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL UNIQUE,
    icon       TEXT    NOT NULL DEFAULT '📦',
    created_at TEXT    NOT NULL DEFAULT (datetime('now'))
);
