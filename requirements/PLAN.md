# Implementation Plan: Accounter — Personal Expense Tracker v1

## Overview

Accounter is a personal expense tracker built as a layered Go monolith. Users log
expenses via a Telegram bot or a web UI (HTMX + Tailwind), view summaries on a
dashboard filtered by time period, and manage categories. A single static Bearer
token secures all endpoints. The application ships as a single Docker container
with SQLite for persistence.

---

## 1. Architecture & Folder Structure

The project follows **feature/domain organisation** — code is grouped by what it
does, not by architectural layer. Shared infrastructure lives in `internal/platform`.

```
accounter/
├── cmd/
│   └── app/
│       └── main.go                    # Entry point: config, DI wiring, server start
│
├── internal/
│   ├── config/
│   │   ├── config.go                  # Env var loading (godotenv non-fatal), Config struct
│   │   └── config_test.go
│   │
│   ├── platform/
│   │   ├── database/
│   │   │   ├── database.go            # Open SQLite connection, apply migrations
│   │   │   └── database_test.go
│   │   ├── migrate/
│   │   │   ├── migrate.go             # Migration runner (go:embed + schema_migrations)
│   │   │   ├── migrate_test.go
│   │   │   └── migrations/            # Embedded .sql files (001_init.sql, etc.)
│   │   │       ├── 001_create_categories.sql
│   │   │       ├── 002_seed_categories.sql
│   │   │       └── 003_create_expenses.sql
│   │   ├── auth/
│   │   │   ├── middleware.go           # Bearer token middleware for chi
│   │   │   └── middleware_test.go
│   │   ├── logger/
│   │   │   ├── logger.go              # slog setup (JSON in prod, text in dev)
│   │   │   └── logger_test.go
│   │   └── server/
│   │       ├── server.go              # HTTP server with graceful shutdown
│   │       └── server_test.go
│   │
│   ├── category/
│   │   ├── model.go                   # Category entity
│   │   ├── repository.go             # CategoryRepository interface
│   │   ├── sqlite_repository.go      # SQLite implementation
│   │   ├── sqlite_repository_test.go
│   │   ├── service.go                # CategoryService (business logic)
│   │   ├── service_test.go
│   │   ├── handler.go                # HTTP handlers (chi)
│   │   └── handler_test.go
│   │
│   ├── expense/
│   │   ├── model.go                   # Expense entity
│   │   ├── repository.go             # ExpenseRepository interface
│   │   ├── sqlite_repository.go      # SQLite implementation
│   │   ├── sqlite_repository_test.go
│   │   ├── service.go                # ExpenseService (business logic)
│   │   ├── service_test.go
│   │   ├── handler.go                # HTTP handlers (chi)
│   │   └── handler_test.go
│   │
│   ├── dashboard/
│   │   ├── handler.go                 # Dashboard page handler (HTML templates)
│   │   ├── handler_test.go
│   │   └── viewmodel.go              # View models for dashboard templates
│   │
│   └── telegram/
│       ├── bot.go                     # Bot setup, long-polling loop
│       ├── bot_test.go
│       ├── commands.go                # Command parsing and dispatch
│       ├── commands_test.go
│       ├── formatter.go              # Format responses for Telegram messages
│       └── formatter_test.go
│
├── web/
│   └── templates/
│       ├── layout.html               # Base layout (Tailwind CDN, HTMX script)
│       ├── dashboard.html            # Dashboard page
│       ├── expenses/
│       │   ├── list.html             # Expense list partial (HTMX target)
│       │   ├── form.html             # Add expense form partial
│       │   └── row.html              # Single expense row partial (HTMX swap)
│       └── categories/
│           ├── list.html             # Category list partial
│           └── form.html             # Add/edit category form partial
│
├── docs/
│   ├── IDEA.md
│   └── PLAN.md                       # This file
│
├── .env                               # Local dev env vars (gitignored values)
├── .env.example                       # Template with all required vars
├── .gitignore
├── Dockerfile
├── mise.toml
├── go.mod
├── go.sum
└── README.md
```

### Package Responsibilities

| Package | Layer | Responsibility |
|---------|-------|----------------|
| `cmd/app` | Composition root | Config loading, DI wiring, start server + bot |
| `internal/config` | Infrastructure | Parse and validate environment variables |
| `internal/platform/database` | Infrastructure | Open SQLite, run migrations |
| `internal/platform/migrate` | Infrastructure | Embed and apply SQL migrations |
| `internal/platform/auth` | Infrastructure | Bearer token middleware |
| `internal/platform/logger` | Infrastructure | slog configuration |
| `internal/platform/server` | Infrastructure | HTTP server lifecycle |
| `internal/category` | Domain + all layers | Category entity, repo, service, handler |
| `internal/expense` | Domain + all layers | Expense entity, repo, service, handler |
| `internal/dashboard` | Presentation | Dashboard HTML rendering |
| `internal/telegram` | Presentation | Telegram bot commands (uses expense/category services) |

### Dependency Flow

```
cmd/app (composition root)
  │
  ├─► internal/config
  ├─► internal/platform/* (database, auth, logger, server)
  │
  ├─► internal/expense
  │     ├── handler  → service → repository (interface)
  │     └── sqlite_repository (implements repository)
  │
  ├─► internal/category
  │     ├── handler  → service → repository (interface)
  │     └── sqlite_repository (implements repository)
  │
  ├─► internal/dashboard
  │     └── handler → expense.Service + category.Service
  │
  └─► internal/telegram
        └── bot → expense.Service + category.Service
```

Key rules:

- **Domain packages never import each other** (expense does not import category).
  When the dashboard needs both, it depends on both services directly.
  The expense *service* may accept a `CategoryChecker` interface (a subset of
  category.Service) to validate that a category exists before creating an expense.
  This interface is defined in the expense package — category satisfies it
  without knowing about it (implicit interface satisfaction).
- **Repository interfaces** are defined in the same package as the domain
  (e.g., `expense.Repository`), following Go's "accept interfaces, return
  structs" idiom.
- **`samber/do`** is used **only** in `cmd/app/main.go`. All other packages
  accept their dependencies as constructor parameters.

---

## 2. Domain Model

### Expense

```go
// internal/expense/model.go
package expense

import "time"

// Expense represents a single financial transaction.
type Expense struct {
    ID          int64     `json:"id"`
    Amount      int64     `json:"amount"`       // Amount in cents (BRL)
    Description string    `json:"description"`
    CategoryID  int64     `json:"category_id"`
    Category    string    `json:"category"`      // Denormalized name (read-only, from JOINs)
    Date        time.Time `json:"date"`          // Date of the expense
    CreatedAt   time.Time `json:"created_at"`
}

// CreateExpenseInput is the input for creating a new expense.
type CreateExpenseInput struct {
    Amount      int64     `json:"amount"`       // Cents
    Description string    `json:"description"`
    CategoryID  int64     `json:"category_id"`
    Date        time.Time `json:"date"`
}

// ListFilter defines filtering and pagination for listing expenses.
type ListFilter struct {
    From     time.Time // Inclusive start date
    To       time.Time // Exclusive end date
    Category *int64    // Optional category ID filter
    Limit    int
    Offset   int
}

// Summary holds aggregated expense data for a time period.
type Summary struct {
    Total        int64           `json:"total"`         // Total in cents
    ByCategory   []CategoryTotal `json:"by_category"`
    ExpenseCount int             `json:"expense_count"`
}

// CategoryTotal is a category with its total amount.
type CategoryTotal struct {
    CategoryID   int64  `json:"category_id"`
    CategoryName string `json:"category_name"`
    Total        int64  `json:"total"`
}

// CategoryChecker validates that a category exists.
// Defined here so expense package doesn't import category package.
// category.Service satisfies this interface implicitly.
type CategoryChecker interface {
    Exists(ctx context.Context, id int64) (bool, error)
}
```

### Category

```go
// internal/category/model.go
package category

import "time"

// Category represents an expense category.
type Category struct {
    ID        int64     `json:"id"`
    Name      string    `json:"name"`
    Icon      string    `json:"icon"`       // Emoji or icon identifier
    CreatedAt time.Time `json:"created_at"`
}

// CreateCategoryInput is the input for creating a new category.
type CreateCategoryInput struct {
    Name string `json:"name"`
    Icon string `json:"icon"`
}

// UpdateCategoryInput is the input for updating an existing category.
type UpdateCategoryInput struct {
    Name *string `json:"name,omitempty"`
    Icon *string `json:"icon,omitempty"`
}
```

### Relationships

```
Category (1) ──────< (N) Expense
   │                      │
   └── id ═══════════════ └── category_id (FK)
```

- Each expense belongs to exactly one category.
- Categories can have many expenses.
- Deleting a category that has expenses is rejected (FK constraint).

### Config

```go
// internal/config/config.go
package config

// Config holds all application configuration.
type Config struct {
    Port                 string // HTTP port (default: "8080")
    DatabasePath         string // SQLite file path (default: "accounter.db")
    BearerToken          string // Auth token (required)
    TelegramToken        string // Telegram bot API token (required)
    TelegramAllowedChat  string // Optional: restrict bot to one chat ID
    LogLevel             string // "debug", "info", "warn", "error" (default: "info")
    Environment          string // "development" or "production" (default: "development")
    Timezone             string // IANA timezone for display (default: "America/Sao_Paulo")
}
```

---

## 3. Database Schema (SQLite Migrations)

### Migration 001: Create categories table

```sql
-- internal/platform/migrate/migrations/001_create_categories.sql

CREATE TABLE IF NOT EXISTS categories (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL UNIQUE,
    icon       TEXT    NOT NULL DEFAULT '📦',
    created_at TEXT    NOT NULL DEFAULT (datetime('now'))
);
```

### Migration 002: Seed default categories

```sql
-- internal/platform/migrate/migrations/002_seed_categories.sql

INSERT OR IGNORE INTO categories (name, icon) VALUES
    ('Alimentação',    '🍔'),
    ('Transporte',     '🚗'),
    ('Moradia',        '🏠'),
    ('Saúde',          '💊'),
    ('Educação',       '📚'),
    ('Lazer',          '🎮'),
    ('Vestuário',      '👕'),
    ('Assinaturas',    '📱'),
    ('Outros',         '📦');
```

### Migration 003: Create expenses table

```sql
-- internal/platform/migrate/migrations/003_create_expenses.sql

CREATE TABLE IF NOT EXISTS expenses (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    amount      INTEGER NOT NULL,
    description TEXT    NOT NULL DEFAULT '',
    category_id INTEGER NOT NULL,
    date        TEXT    NOT NULL,
    created_at  TEXT    NOT NULL DEFAULT (datetime('now')),

    FOREIGN KEY (category_id) REFERENCES categories(id)
);

CREATE INDEX IF NOT EXISTS idx_expenses_date ON expenses(date);
CREATE INDEX IF NOT EXISTS idx_expenses_category_id ON expenses(category_id);
```

### Schema Migrations Table (created by the migration runner)

```sql
-- Created automatically by the migrate package
CREATE TABLE IF NOT EXISTS schema_migrations (
    version    TEXT    NOT NULL UNIQUE,
    applied_at TEXT    NOT NULL DEFAULT (datetime('now'))
);
```

### Migration Runner Design

```go
// internal/platform/migrate/migrate.go
package migrate

import (
    "database/sql"
    "embed"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Run applies all pending migrations in order.
// 1. Create schema_migrations table if not exists
// 2. Read all .sql files from embedded FS
// 3. Sort by filename (lexicographic = version order)
// 4. For each migration not in schema_migrations:
//    a. Begin transaction
//    b. Execute SQL
//    c. Insert version into schema_migrations
//    d. Commit (or rollback on error)
// 5. Return nil or first error
func Run(db *sql.DB) error { /* ... */ }
```

---

## 4. API Surface

### Authentication

All API and dashboard routes require an `Authorization: Bearer <token>` header.
The Telegram bot authenticates via its own token (Telegram API validates the
source). A `/health` endpoint is unauthenticated.

### REST API Endpoints

#### Expenses

| Method | Path | Description | Request | Response |
|--------|------|-------------|---------|----------|
| `POST` | `/api/expenses` | Create expense | `CreateExpenseInput` JSON | `201` + `Expense` JSON |
| `GET` | `/api/expenses` | List expenses (filtered) | Query: `from`, `to`, `category_id`, `limit`, `offset` | `200` + `[]Expense` JSON |
| `GET` | `/api/expenses/{id}` | Get single expense | — | `200` + `Expense` JSON |
| `DELETE` | `/api/expenses/{id}` | Delete expense | — | `204` No Content |
| `GET` | `/api/expenses/summary` | Aggregated summary | Query: `from`, `to` | `200` + `Summary` JSON |

#### Categories

| Method | Path | Description | Request | Response |
|--------|------|-------------|---------|----------|
| `GET` | `/api/categories` | List all categories | — | `200` + `[]Category` JSON |
| `POST` | `/api/categories` | Create category | `CreateCategoryInput` JSON | `201` + `Category` JSON |
| `PUT` | `/api/categories/{id}` | Update category | `UpdateCategoryInput` JSON | `200` + `Category` JSON |
| `DELETE` | `/api/categories/{id}` | Delete category | — | `204` (or `409` if in use) |

### Web UI Routes (HTMX)

| Method | Path | Description | Response |
|--------|------|-------------|----------|
| `GET` | `/` | Redirect to dashboard | `302` |
| `GET` | `/dashboard` | Main dashboard page | HTML (full page) |
| `GET` | `/dashboard/expenses` | Expense list partial | HTML fragment |
| `POST` | `/dashboard/expenses` | Create expense (form) | HTML fragment (new row) |
| `DELETE` | `/dashboard/expenses/{id}` | Delete expense | `200` empty |
| `GET` | `/dashboard/categories` | Category list partial | HTML fragment |
| `POST` | `/dashboard/categories` | Create category (form) | HTML fragment |

### Health Check

| Method | Path | Description | Response |
|--------|------|-------------|----------|
| `GET` | `/health` | Liveness probe | `200` `{"status":"ok"}` |

### Request/Response Shapes

```jsonc
// POST /api/expenses — Request
{
    "amount": 2550,           // R$ 25.50 in cents
    "description": "Almoço",
    "category_id": 1,
    "date": "2026-03-16"      // Optional, defaults to today
}

// POST /api/expenses — Response (201)
{
    "id": 42,
    "amount": 2550,
    "description": "Almoço",
    "category_id": 1,
    "category": "Alimentação",
    "date": "2026-03-16",
    "created_at": "2026-03-16T12:30:00Z"
}

// GET /api/expenses/summary?from=2026-03-01&to=2026-04-01 — Response (200)
{
    "total": 185000,
    "expense_count": 47,
    "by_category": [
        { "category_id": 1, "category_name": "Alimentação", "total": 85000 },
        { "category_id": 2, "category_name": "Transporte", "total": 35000 }
    ]
}

// Error Response (4xx/5xx)
{
    "error": "amount must be greater than zero"
}
```

### Router Structure (chi)

```go
r := chi.NewRouter()

// Global middleware
r.Use(middleware.RequestID)
r.Use(middleware.RealIP)
r.Use(slogMiddleware(logger)) // custom slog request logging middleware
r.Use(middleware.Recoverer)

// Public
r.Get("/health", healthHandler)

// Protected routes
r.Group(func(r chi.Router) {
    r.Use(auth.BearerMiddleware(cfg.BearerToken))

    // JSON API
    r.Route("/api", func(r chi.Router) {
        r.Route("/expenses", func(r chi.Router) {
            r.Post("/", expenseHandler.Create)
            r.Get("/", expenseHandler.List)
            r.Get("/summary", expenseHandler.Summary)
            r.Get("/{id}", expenseHandler.Get)
            r.Delete("/{id}", expenseHandler.Delete)
        })
        r.Route("/categories", func(r chi.Router) {
            r.Get("/", categoryHandler.List)
            r.Post("/", categoryHandler.Create)
            r.Put("/{id}", categoryHandler.Update)
            r.Delete("/{id}", categoryHandler.Delete)
        })
    })

    // Web UI (HTMX)
    r.Get("/", redirectToDashboard)
    r.Route("/dashboard", func(r chi.Router) {
        r.Get("/", dashboardHandler.Index)
        r.Get("/expenses", dashboardHandler.ExpenseList)
        r.Post("/expenses", dashboardHandler.CreateExpense)
        r.Delete("/expenses/{id}", dashboardHandler.DeleteExpense)
        r.Get("/categories", dashboardHandler.CategoryList)
        r.Post("/categories", dashboardHandler.CreateCategory)
    })
})
```

---

## 5. Telegram Bot Flow

### Integration Architecture

```
Telegram API
     │
     │ (long polling via go-telegram-bot-api)
     ▼
telegram.Bot
     │
     ├── parses command + args
     │
     ├── /add ────► expense.Service.Create()
     ├── /list ───► expense.Service.List()
     ├── /today ──► expense.Service.Summary(today)
     ├── /month ──► expense.Service.Summary(thisMonth)
     ├── /categories ► category.Service.List()
     │
     ├── formats response (formatter.go)
     │
     └── sends reply via Telegram API
```

The Telegram bot uses the **exact same service layer** as the HTTP handlers.
It is purely a presentation adapter — it parses text commands into service calls
and formats results back into text messages.

### Bot Lifecycle

```go
// internal/telegram/bot.go
package telegram

type Bot struct {
    api             *tgbotapi.BotAPI
    expenseService  ExpenseService    // Interface defined in this package
    categoryService CategoryService   // Interface defined in this package
    allowedChatID   int64             // 0 = allow all
    logger          *slog.Logger
}

// Service interfaces defined in telegram package (narrow, only what bot needs)
type ExpenseService interface {
    Create(ctx context.Context, input expense.CreateExpenseInput) (expense.Expense, error)
    List(ctx context.Context, filter expense.ListFilter) ([]expense.Expense, error)
    Summary(ctx context.Context, from, to time.Time) (expense.Summary, error)
}

type CategoryService interface {
    List(ctx context.Context) ([]category.Category, error)
    GetByName(ctx context.Context, name string) (category.Category, error)
}

func NewBot(token string, es ExpenseService, cs CategoryService, allowedChat int64, logger *slog.Logger) (*Bot, error) {
    api, err := tgbotapi.NewBotAPI(token)
    if err != nil {
        return nil, fmt.Errorf("telegram bot init: %w", err)
    }
    return &Bot{api: api, expenseService: es, categoryService: cs, allowedChatID: allowedChat, logger: logger}, nil
}

// Start begins long-polling. Blocks until ctx is cancelled.
func (b *Bot) Start(ctx context.Context) error {
    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60
    updates := b.api.GetUpdatesChan(u)

    for {
        select {
        case <-ctx.Done():
            b.api.StopReceivingUpdates()
            return ctx.Err()
        case update := <-updates:
            if update.Message == nil || !update.Message.IsCommand() {
                continue
            }
            if b.allowedChatID != 0 && update.Message.Chat.ID != b.allowedChatID {
                continue // silently ignore unauthorized chats
            }
            b.handleCommand(ctx, update.Message)
        }
    }
}
```

### Command Parsing Convention

For `/add`, the convention is:

```
/add <amount> <category> <description...>
```

- `amount`: decimal number (e.g., `25.50` or `25,50`), converted to cents
- `category`: single word, matched case-insensitively against category names
- `description`: everything remaining (supports multi-word)

This ordering (category before description) avoids the ambiguity of trying to
detect where a multi-word description ends and the category begins.

Example: `/add 25.50 Alimentação Almoço no restaurante do João`

### Telegram Bot Authentication

The bot does **not** use the Bearer token. Instead:

- Telegram API validates that updates come from Telegram servers.
- Optionally, restrict the bot to a specific chat ID (configured via env var
  `TELEGRAM_ALLOWED_CHAT_ID`) to prevent unauthorized users from interacting
  with the bot.

---

## 6. Task Breakdown (TDD-First)

Each task produces working, tested code. Tasks are ordered by dependency.

### Phase 1: Foundation (Infrastructure)

> **Goal:** Project compiles, config loads, database connects, migrations run.

status: Done

**1. Initialize Go module and dependencies**

- 1.1. Run `go mod init github.com/gabrigas/accounter`
- 1.2. Add dependencies: `chi/v5`, `modernc.org/sqlite`, `samber/do/v2`,
     `go-telegram-bot-api/v5`, `joho/godotenv`
- 1.3. Create `cmd/app/main.go` with minimal `func main()`
- 1.4. Verify `go build ./cmd/app` succeeds
- 1.5. Create `.gitignore` (binaries, `.db`, `.env`)
- Dependencies: None
- Risk: Low

**2. Config loading** (`internal/config/`)

- 2.1. Write test: loading from env vars produces correct Config struct
- 2.2. Write test: missing required vars (`BEARER_TOKEN`, `TELEGRAM_TOKEN`) returns error
- 2.3. Write test: defaults are applied for optional vars (port, log level, etc.)
- 2.4. Implement `Load() (Config, error)` using `os.Getenv` + `godotenv` (non-fatal)
- 2.5. Create `.env.example` with all variables documented
- Dependencies: Task 1
- Risk: Low

**3. Logger setup** (`internal/platform/logger/`)

- 3.1. Write test: dev environment produces text handler
- 3.2. Write test: prod environment produces JSON handler
- 3.3. Write test: log level string is parsed correctly
- 3.4. Implement `New(level, environment string) *slog.Logger`
- Dependencies: None
- Risk: Low

**4. Migration runner** (`internal/platform/migrate/`)

- 4.1. Write test: applies all migrations to fresh in-memory SQLite
- 4.2. Write test: skips already-applied migrations (idempotent)
- 4.3. Write test: records applied versions in schema_migrations
- 4.4. Write test: rolls back transaction on invalid SQL (no partial apply)
- 4.5. Implement `Run(db *sql.DB) error` with `go:embed`
- 4.6. Create 3 migration SQL files (001, 002, 003)
- Dependencies: Task 1 (needs `modernc.org/sqlite`)
- Risk: Medium — must handle SQLite dialect for dates, FK pragma

**5. Database connection** (`internal/platform/database/`)

- 5.1. Write test: opens in-memory SQLite and runs migrations successfully
- 5.2. Write test: enables WAL mode and foreign keys
- 5.3. Implement `Open(path string) (*sql.DB, error)` with pragmas:
     `PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;`
- Dependencies: Task 4
- Risk: Low

### Phase 2: Category Domain (Vertical Slice)

> **Goal:** Categories fully CRUD-able via service and repository layers.

status: done

**6. Category model** (`internal/category/model.go`)

- 6.1. Define `Category`, `CreateCategoryInput`, `UpdateCategoryInput` structs
- Dependencies: None
- Risk: Low

**7. Category repository** (`internal/category/`)

- 7.1. Define `Repository` interface in `repository.go`
- 7.2. Write test: `Create` inserts category and returns it with ID
- 7.3. Write test: `Create` with duplicate name returns error
- 7.4. Write test: `List` returns all categories ordered by name
- 7.5. Write test: `GetByID` returns category or `ErrNotFound`
- 7.6. Write test: `GetByName` returns category (case-insensitive) or `ErrNotFound`
- 7.7. Write test: `Exists` returns true/false by ID
- 7.8. Write test: `Update` modifies name and/or icon
- 7.9. Write test: `Delete` removes category
- 7.10. Write test: `Delete` with FK constraint (category has expenses) → error
- 7.11. Implement `SQLiteRepository` for all methods
- Dependencies: Tasks 5, 6
- Risk: Low

**8. Category service** (`internal/category/`)

- 8.1. Define `Service` interface in `service.go`
- 8.2. Write test: `Create` with valid input delegates to repo and returns result
- 8.3. Write test: `Create` with empty name → validation error
- 8.4. Write test: `Create` with empty icon → defaults to '📦'
- 8.5. Write test: `List` returns all categories from repo
- 8.6. Write test: `GetByID` delegates to repo
- 8.7. Write test: `GetByName` delegates to repo
- 8.8. Write test: `Exists` delegates to repo
- 8.9. Write test: `Update` with valid input delegates to repo
- 8.10. Write test: `Delete` delegates to repo
- 8.11. Implement `DefaultService` struct (tests use mock repo)
- Dependencies: Tasks 6, 7
- Risk: Low

### Phase 3: Expense Domain (Vertical Slice)

> **Goal:** Expenses fully CRUD-able with filtering and summaries.

status: done

**9. Expense model** (`internal/expense/model.go`)

- 9.1. Define `Expense`, `CreateExpenseInput`, `ListFilter`, `Summary`,
     `CategoryTotal`, `CategoryChecker` types
- Dependencies: None
- Risk: Low

**10. Expense repository** (`internal/expense/`)

- 10.1. Define `Repository` interface in `repository.go`
- 10.2. Write test: `Create` inserts expense with valid category → returns with ID and joined category name
- 10.3. Write test: `Create` with invalid category_id → FK constraint error
- 10.4. Write test: `GetByID` returns expense with category name or `ErrNotFound`
- 10.5. Write test: `List` with date range filter returns correct expenses
- 10.6. Write test: `List` with category filter
- 10.7. Write test: `List` with pagination (limit/offset)
- 10.8. Write test: `List` ordered by date descending (newest first)
- 10.9. Write test: `Delete` removes expense or returns `ErrNotFound`
- 10.10. Write test: `Summary` returns total, count, and per-category breakdown
- 10.11. Write test: `Summary` with no expenses returns zero total and empty breakdown
- 10.12. Implement `SQLiteRepository`
- Dependencies: Tasks 5, 7 (needs categories seeded for FK), 9
- Risk: Medium — date filtering SQL, aggregation with GROUP BY

**11. Expense service** (`internal/expense/`)

- 11.1. Define `Service` interface in `service.go`
- 11.2. Write test: `Create` with valid input delegates to repo
- 11.3. Write test: `Create` with amount ≤ 0 → validation error
- 11.4. Write test: `Create` validates category exists via `CategoryChecker`
- 11.5. Write test: `Create` with non-existent category → error
- 11.6. Write test: `Create` with zero date → defaults to today
- 11.7. Write test: `List` builds filter and delegates to repo
- 11.8. Write test: `Summary` delegates to repo
- 11.9. Write test: `Delete` delegates to repo
- 11.10. Implement `DefaultService` (tests use mock repo + mock CategoryChecker)
- Dependencies: Tasks 8, 9, 10
- Risk: Low

### Phase 4: Auth Middleware

> **Goal:** Bearer token auth protects all non-public routes.

status: done

**12. Auth middleware** (`internal/platform/auth/`)

- 12.1. Write test: request with valid `Authorization: Bearer <token>` → next handler called
- 12.2. Write test: request without Authorization header → 401 JSON error
- 12.3. Write test: request with wrong token → 401 JSON error
- 12.4. Write test: request with malformed header (no "Bearer " prefix) → 401
- 12.5. Write test: request with empty token value → 401
- 12.6. Implement `BearerMiddleware(token string) func(http.Handler) http.Handler`
- Dependencies: None (can be done in parallel with any phase)
- Risk: Low

### Phase 5: HTTP Handlers (API)

> **Goal:** Full REST API for expenses and categories.

status: done

**13. Category HTTP handler** (`internal/category/handler.go`)

- 13.1. Write test: `POST /api/categories` with valid JSON → 201 + created category
- 13.2. Write test: `POST /api/categories` with invalid JSON → 400
- 13.3. Write test: `POST /api/categories` with empty name → 400
- 13.4. Write test: `GET /api/categories` → 200 + JSON array
- 13.5. Write test: `PUT /api/categories/{id}` with valid JSON → 200 + updated
- 13.6. Write test: `PUT /api/categories/{id}` not found → 404
- 13.7. Write test: `DELETE /api/categories/{id}` → 204
- 13.8. Write test: `DELETE /api/categories/{id}` in use → 409 conflict
- 13.9. Implement `Handler` struct with `Routes() chi.Router`
- 13.10. **(Missing)** Add integration tests for category endpoints (with build tag)
- Dependencies: Task 8
- Risk: Low

**14. Expense HTTP handler** (`internal/expense/handler.go`)

- 14.1. Write test: `POST /api/expenses` with valid JSON → 201 + created expense
- 14.2. Write test: `POST /api/expenses` with amount=0 → 400
- 14.3. Write test: `POST /api/expenses` with invalid category → 400
- 14.4. Write test: `GET /api/expenses` without filters → 200 + default list
- 14.5. Write test: `GET /api/expenses?from=2026-03-01&to=2026-04-01` → filtered list
- 14.6. Write test: `GET /api/expenses/{id}` → 200 + expense
- 14.7. Write test: `GET /api/expenses/{id}` not found → 404
- 14.8. Write test: `GET /api/expenses/summary?from=...&to=...` → 200 + summary
- 14.9. Write test: `DELETE /api/expenses/{id}` → 204
- 14.10. Implement `Handler` struct with `Routes() chi.Router`
- 14.11. **(Missing)** Add integration tests for expense endpoints (with build tag)
- Dependencies: Task 11
- Risk: Low

**14.a OpenAPI Documentation (swaggo)**

- 14.a.1. Add swaggo annotations to all API handlers (using `github.com/swaggo/swag`)
- 14.a.2. Generate OpenAPI 2.0/3.0 spec files
- 14.a.3. Serve Swagger UI at `/swagger` using `github.com/swaggo/http-swagger` (recommended for chi)
- Dependencies: Phase 5
- Risk: Low

### Phase 6: Web Dashboard (HTMX) — UI Rework

> **Goal:** Autonomously implement a visually appealing, functional, and complete dashboard following the "Financial Sanctuary" aesthetic (see `DESIGN.md`).

status: todo

**15. HTML templates (Reworked)** (`web/templates/`)

- 15.1. Update `layout.html` with Tailwind configuration, fonts (Outfit & Inter), and base styles.
- 15.2. Redesign `dashboard.html` to implement the "Financial Sanctuary" layout:
  - Multi-column layout with sidebar and main content area.
  - Glassmorphic components (`.glass` class).
  - High-end typography and spacing.
- 15.3. Create `summary.html` with animated summary cards and radial gradients.
- 15.4. Rework `expenses/list.html` and `expenses/row.html` for the new aesthetic.
- 15.5. Rework `expenses/form.html` to be a modern, accessible form with validation feedback.
- 15.6. Rework `categories/list.html` with a grid of icon-based cards.
- 15.7. Add loading indicators (`hx-indicator`) and transition animations (`.animate-fade-in`).
- Dependencies: Phase 3
- Risk: Medium — achieving the desired visual polish with HTMX/Tailwind.

**16. Dashboard view models (Updated)** (`internal/dashboard/viewmodel.go`)

- 16.1. Update `DashboardData` to include additional fields for the new UI (e.g., active menu item, user initials).
- 16.2. Add support for more granular formatting (e.g., separating currency symbols from values).
- 16.3. Add support for category-specific colors/icons in the view model.
- Dependencies: Tasks 9, 6
- Risk: Low

**17. Dashboard handler (Reworked)** (`internal/dashboard/handler.go`)

- 17.1. Update handler to support new template structure and HTMX triggers.
- 17.2. Ensure proper error handling and flash messages using HTMX headers.
- 17.3. Implement server-side filtering logic for the new period controls.
- Dependencies: Tasks 11, 8, 15, 16
- Risk: Medium — complex UI state management with HTMX.

### Phase 7: Telegram Bot

> **Goal:** Users can log and query expenses via Telegram.

status: todo

**18. Telegram formatter** (`internal/telegram/formatter.go`)

- 18.1. Write test: format single expense → "🍔 R$ 25,50 — Almoço (16/03)"
- 18.2. Write test: format expense list → numbered list
- 18.3. Write test: format summary → total + per-category lines
- 18.4. Write test: format category list → emoji + name per line
- 18.5. Write test: format error message → "❌ " prefix
- 18.6. Implement all formatters (plain text + emoji)
- Dependencies: Tasks 9, 6
- Risk: Low

**19. Telegram command parser** (`internal/telegram/commands.go`)

- 19.1. Write test: `/add 25.50 Alimentação Almoço no restaurante` → valid input (amount=2550, cat="Alimentação", desc="Almoço no restaurante")
- 19.2. Write test: `/add 25,50 Alimentação Almoço` (comma decimal) → valid
- 19.3. Write test: `/add` with missing args → error message
- 19.4. Write test: `/add 0 Alimentação desc` → error (positive amount required)
- 19.5. Write test: `/add 25.50 UnknownCategory desc` → error (category not found)
- 19.6. Write test: `/list` calls service.List with limit=5
- 19.7. Write test: `/today` calls service.Summary with today's date range
- 19.8. Write test: `/month` calls service.Summary with current month range
- 19.9. Write test: `/categories` returns formatted category list
- 19.10. Write test: unknown command → help message listing available commands
- 19.11. Implement command dispatcher with mock services in tests
- Dependencies: Tasks 11, 8, 18
- Risk: Medium — parsing edge cases

**20. Telegram bot lifecycle** (`internal/telegram/bot.go`)

- 20.1. Write test: bot processes updates from a channel and dispatches commands
- 20.2. Write test: bot stops gracefully on context cancellation
- 20.3. Write test: bot ignores non-command messages
- 20.4. Write test: bot rejects updates from unauthorized chat IDs (if configured)
- 20.5. Implement `Bot.Start(ctx)` with long-polling loop
- Dependencies: Task 19
- Risk: Medium — testing requires mock update channel

### Phase 8: Wiring & Server

> **Goal:** Everything starts together with graceful shutdown.

status: done

**21. HTTP server** (`internal/platform/server/`)

- 21.1. Write test: server starts and responds to health check
- 21.2. Write test: server shuts down gracefully on context cancel
- 21.3. Implement `Server` struct with `Start(ctx)` (blocks, returns on shutdown)
- Dependencies: None
- Risk: Low

**22. DI wiring** (`cmd/app/main.go`)

- 22.1. Wire config → logger → database (with migrations)
- 22.2. Wire category: repository → service → handler
- 22.3. Wire expense: repository → service → handler
- 22.4. Wire dashboard handler
- 22.5. Wire telegram bot
- 22.6. Build chi router with all routes + middleware
- 22.7. Wire HTTP server
- 22.8. Start bot goroutine + server, wait for SIGINT/SIGTERM
- 22.9. Graceful shutdown: cancel context → bot stops + server drains
- Dependencies: All previous tasks
- Risk: Medium — correct shutdown orchestration

### Phase 9: Docker & Tooling

> **Goal:** `docker build && docker run` works end-to-end.

status: in progress

**23. Fix and finalize Dockerfile**

- 23.1. Fix binary name mismatch (`go-worker` vs CMD `/app/app` → use `accounter`)
- 23.2. Update Go version to match actual (1.26)
- 23.3. Add `VOLUME ["/data"]` for SQLite persistence
- 23.4. Add `EXPOSE 8080`
- 23.5. Add `HEALTHCHECK` instruction
- 23.6. Copy templates if using filesystem-based (or remove if `go:embed`)
- 23.7. Test full Docker build and run
- Dependencies: Task 22
- Risk: Low

**24. mise tasks** (`mise.toml`)

- 24.1. Add `test` task: `go test ./... -race -cover`
- 24.2. Add `lint` task: `go vet ./...`
- 24.3. Add `coverage` task: `go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out`
- 24.4. Add `docker-build` task
- 24.5. Add `docker-run` task
- 24.6. Update existing `start` task if needed
- Dependencies: Task 23
- Risk: Low

### Phase 10: Documentation & Polish

status: in progress

**25. Documentation**

- 25.1. Update README with: project overview, setup instructions, env vars, API docs
- 25.2. Document Telegram bot setup (BotFather, token, chat ID)
- 25.3. Document Docker usage (build, run, volume mount)
- 25.4. Verify `.gitignore` covers: binaries, `.db`, `.db-wal`, `.db-shm`, `.env`
- Dependencies: All tasks
- Risk: Low

---

## 7. Dependency Wiring

### samber/do in main.go

`samber/do` serves as the DI container **only** in `cmd/app/main.go`. Every
other package uses plain constructor injection.

```go
// Simplified example — see full version in Section 5 (Task 22)
injector := do.New()

// Each Provide registers a factory function
do.Provide(injector, func(i do.Injector) (*config.Config, error) {
    return config.Load()
})

do.Provide(injector, func(i do.Injector) (category.Repository, error) {
    db := do.MustInvoke[*sql.DB](i)
    return category.NewSQLiteRepository(db), nil
})

do.Provide(injector, func(i do.Injector) (category.Service, error) {
    repo := do.MustInvoke[category.Repository](i)
    log := do.MustInvoke[*slog.Logger](i)
    return category.NewService(repo, log), nil
})

// Lazy resolution — nothing is created until first MustInvoke
srv := do.MustInvoke[*server.Server](injector)
```

### Tests Bypass DI — Manual Wiring

```go
// Service tests use mock repositories
func TestExpenseService_Create(t *testing.T) {
    repo := &mockExpenseRepository{}
    checker := &mockCategoryChecker{exists: true}
    logger := slog.New(slog.NewTextHandler(io.Discard, nil))

    svc := expense.NewService(repo, checker, logger)

    input := expense.CreateExpenseInput{
        Amount:      2550,
        Description: "Almoço",
        CategoryID:  1,
    }
    got, err := svc.Create(context.Background(), input)
    require.NoError(t, err)
    assert.Equal(t, int64(2550), got.Amount)
}
```

```go
// Repository tests use real in-memory SQLite
func setupTestDB(t *testing.T) *sql.DB {
    t.Helper()
    db, err := database.Open(":memory:")
    require.NoError(t, err)
    t.Cleanup(func() { db.Close() })
    return db
}

func TestSQLiteRepository_Create(t *testing.T) {
    db := setupTestDB(t)
    repo := expense.NewSQLiteRepository(db)
    // ... test against real SQLite with migrations applied
}
```

```go
// Handler tests use mock services
func TestExpenseHandler_Create(t *testing.T) {
    svc := &mockExpenseService{}
    logger := slog.New(slog.NewTextHandler(io.Discard, nil))
    handler := expense.NewHandler(svc, logger)

    body := `{"amount":2550,"description":"Test","category_id":1}`
    req := httptest.NewRequest("POST", "/api/expenses", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()

    handler.Create(rec, req)

    assert.Equal(t, http.StatusCreated, rec.Code)
}
```

**Principle:** Every constructor is `func New...(dep1, dep2, ...) *Type`.
No package ever imports `samber/do`. Tests compose objects directly.

---

## 8. Docker Strategy

### Fixed Dockerfile

The existing Dockerfile has several issues to fix:

| Issue | Current | Fixed |
|-------|---------|-------|
| Binary name mismatch | Builds `go-worker`, CMD runs `/app/app` | Build and run `accounter` |
| Go version | `golang:1.25-alpine` (may not exist) | `golang:1.26-alpine` |
| No volume for data | — | `VOLUME ["/data"]` |
| No health check | — | `HEALTHCHECK` added |
| No port exposed | — | `EXPOSE 8080` |
| Ubuntu base image tag | `ubuntu:oracular` | `ubuntu:noble` (LTS) |

### Environment Variable Strategy

| Variable | Required | Default | Notes |
|----------|----------|---------|-------|
| `BEARER_TOKEN` | Yes | — | Auth token for API/dashboard |
| `TELEGRAM_TOKEN` | Yes | — | Bot token from BotFather |
| `PORT` | No | `8080` | HTTP listen port |
| `DATABASE_PATH` | No | `accounter.db` | Use `/data/accounter.db` in Docker |
| `LOG_LEVEL` | No | `info` | debug/info/warn/error |
| `ENVIRONMENT` | No | `development` | development/production |
| `TIMEZONE` | No | `America/Sao_Paulo` | For display formatting |
| `TELEGRAM_ALLOWED_CHAT_ID` | No | — | Restrict bot to one chat |

### Volume for SQLite Data

- Container path: `/data/`
- SQLite creates 3 files: `accounter.db`, `accounter.db-wal`, `accounter.db-shm`
- Use a named Docker volume or bind mount for persistence
- **Never** put the DB file in the image layer

### Docker Compose (Optional, for convenience)

```yaml
# docker-compose.yml
services:
  accounter:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - accounter-data:/data
    environment:
      - BEARER_TOKEN=${BEARER_TOKEN}
      - TELEGRAM_TOKEN=${TELEGRAM_TOKEN}
      - DATABASE_PATH=/data/accounter.db
      - ENVIRONMENT=production
      - LOG_LEVEL=info
    restart: unless-stopped

volumes:
  accounter-data:
```

---

## 9. Open Questions & Risks

### Must Resolve Before Coding

| # | Question | Recommendation | Impact |
|---|----------|----------------|--------|
| 1 | **Telegram command argument order** — How to handle multi-word descriptions in `/add`? | Use `/add <amount> <category> <description...>` — category is always one word (matches category name), everything after is description. | Medium — affects parser design |
| 2 | **Timezone handling** — Dashboard displays dates in which timezone? | Store all dates in UTC (SQLite `datetime('now')`). Add `TIMEZONE` env var (default `America/Sao_Paulo`). Convert on display only. | Low — affects all date formatting |
| 3 | **Templates: embed vs filesystem** — Should templates be embedded with `go:embed` or loaded from filesystem? | Use `go:embed` for production (single binary, scratch image). Optionally fall back to filesystem in `development` mode for hot-reload during development. | Low — DX convenience |

### Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| **SQLite concurrent writes** | Low (single user) | Medium | WAL mode enabled. Single user = non-issue. |
| **modernc.org/sqlite with CGO_ENABLED=0** | Low | High | `modernc.org/sqlite` is pure Go — confirmed compatible. Verify in Task 1. |
| **Telegram token exposure in logs** | Medium | High | Never log tokens. Mask in config display. Add `.env` to `.gitignore`. |
| **Amount precision with float parsing** | Medium | Medium | Parse user input as string → `strconv.ParseFloat` → multiply by 100 → `int64`. Never store floats. |
| **Template hot-reload DX** | Low | Low | `go:embed` in prod, filesystem in dev via `ENVIRONMENT` check. |
| **go-telegram-bot-api library maintenance** | Low | Low | Library is stable. Minimal surface used. Easy to swap later. |
| **Dockerfile scratch image + templates** | Low | Medium | If using `go:embed`, no issue. If filesystem templates, need non-scratch image. Prefer `go:embed`. |

### Deferred to Post-v1

- Investment tracking tab (new domain, new tables, new UI)
- Google Sheets export (new service, Google API integration)
- Swappable persistence (Repository interfaces already enable this)
- Multi-user support (users table, per-user tokens, query scoping)
- Expense editing (`PUT /api/expenses/{id}`)
- Recurring expenses
- Budget/spending limits per category

---

## 10. Success Criteria for v1

- [ ] `go test ./... -race` passes with ≥80% coverage
- [ ] `docker build` produces a working image under 30MB
- [ ] `docker run` starts the app, runs migrations, serves dashboard
- [ ] Bearer token authentication protects all API and dashboard endpoints
- [ ] `/health` endpoint responds without authentication
- [ ] Can create, list, get, delete expenses via JSON API
- [ ] Can create, list, update, delete categories via JSON API
- [ ] `GET /api/expenses/summary` returns correct totals and per-category breakdown
- [ ] Dashboard renders expenses filtered by day/month/year/custom period
- [ ] Dashboard shows per-category spending breakdown
- [ ] Telegram `/add` creates an expense and confirms
- [ ] Telegram `/list`, `/today`, `/month`, `/categories` return correct data
- [ ] Structured JSON logging in production, text in development
- [ ] Graceful shutdown (HTTP server + Telegram bot) on SIGINT/SIGTERM
- [ ] SQLite data persists across container restarts via volume mount
- [ ] All amounts stored as integer cents (no floating point)
