# Accounter — Agent Development Guide

This guide provides agentic coding agents with essential information for working on the Accounter personal expense tracker project.

## Project Overview

Accounter is a personal expense tracker built as a **layered Go monolith** using **feature/domain organization**. Users log expenses via a Telegram bot or web UI (HTMX + Tailwind), view summaries on a dashboard, and manage categories. A single static Bearer token secures all endpoints. Ships as a single Docker container with SQLite for persistence.

**Key Technologies:** Go 1.26, SQLite (modernc.org/sqlite), chi router, samber/do DI, HTMX, Tailwind CSS, go-telegram-bot-api

---

## Development Workflow

**CRITICAL: Follow this workflow for ALL feature development and bug fixes**

### 1. Write Tests First (TDD)

Before implementing any feature or fix:

- **Write failing tests** that define the expected behavior
- Use table-driven tests with comprehensive test cases
- Cover happy paths AND edge cases (validation, errors, boundary conditions)
- Repository tests: use real in-memory SQLite
- Service tests: use mock repositories
- Handler tests: use mock services

```bash
# Create test file first
# Example: internal/expense/service_test.go

# Run tests to verify they fail (RED phase)
go test -v -race ./internal/expense -run TestExpenseService_Create
```

### 2. Implementation

After tests are written:

- Implement the minimal code to make tests pass
- Follow all code style guidelines (MixedCaps, error handling, etc.)
- Keep functions small and focused
- Use early returns to minimize indentation
- Wrap errors with context using `fmt.Errorf("context: %w", err)`

```bash
# Implement feature in production code
# Example: internal/expense/service.go
```

### 3. Verify Tests Pass

Run tests to confirm implementation is correct:

```bash
# Run all tests with race detection and coverage
go test ./... -race -coverprofile=coverage.out

# Run specific package tests
go test -v -race ./internal/expense/...

# View coverage report
go tool cover -html=coverage.out -o coverage.html
```

**Requirements:**
- ✅ All tests must pass
- ✅ No race conditions detected
- ✅ Coverage should be maintained or improved

### 4. Quality Checks

Run linter and formatter before committing:

```bash
# Format code
go fmt ./...
go vet ./...

# Run linter (MUST pass with zero warnings)
golangci-lint run ./...

# Auto-fix simple issues
golangci-lint run --fix ./...

# Tidy dependencies
go mod tidy
```

**Requirements:**
- ✅ Code must be formatted (`go fmt`)
- ✅ No `go vet` warnings
- ✅ Zero `golangci-lint` errors/warnings
- ✅ `go.mod` and `go.sum` are tidy

### 5. Commit (After User Permission)

**NEVER commit without explicit user approval**

After all checks pass, request permission to commit:

```bash
# Check git status
git status

# View diff
git diff

# Stage changes (after user approval)
git add .

# Commit with descriptive message (after user approval)
git commit -m "feat(expense): add validation for negative amounts

- Add ErrInvalidAmount sentinel error
- Validate amount > 0 in service layer
- Add comprehensive test cases for amount validation"
```

**Commit Message Format:**
- **Type**: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`
- **Scope**: domain name (`expense`, `category`, `telegram`, etc.)
- **Subject**: imperative mood, lowercase, no period
- **Body** (optional): explain WHY, not WHAT

### Quick Workflow Reference

```bash
# 1. TDD: Write tests first
vim internal/domain/feature_test.go
go test -v ./internal/domain -run TestFeature  # Should FAIL

# 2. Implementation
vim internal/domain/feature.go
go test -v ./internal/domain -run TestFeature  # Should PASS

# 3. Verify all tests
go test ./... -race -coverprofile=coverage.out

# 4. Quality checks
go fmt ./...
go vet ./...
golangci-lint run ./...
go mod tidy

# 5. Commit (with user permission only)
git status
git diff
# [User approves]
git add .
git commit -m "type(scope): description"
```

**Using mise tasks:**

```bash
# Run tests
mise run test

# Run linter
mise run lint

# Format code
mise run fmt
```

---

## Build, Lint & Test Commands

### Essential Commands

```bash
# Run all tests with race detection and coverage
go test ./... -race -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# Run a single test
go test -v -race ./internal/expense -run TestExpenseService_Create

# Run a single test case within a table-driven test
go test -v ./internal/expense -run TestExpenseService_Create/valid_input

# Run tests in a specific package
go test -v -race ./internal/expense/...

# Run integration tests (with build tag)
go test -v -race -tags=integration ./...

# Run short tests (exclude long-running)
go test -v -short ./...

# Build the application
go build -v -ldflags "-X main.Version=$(git describe --tags --always --dirty)" -o bin/accounter ./cmd/app

# Lint with golangci-lint
golangci-lint run ./...
golangci-lint run --fix ./...

# Format code
go fmt ./...
go vet ./...

# Check for outdated dependencies
go list -u -m all

# Audit dependencies for vulnerabilities
govulncheck ./...

# Run application locally
go run ./cmd/app

# Clean build artifacts
rm -rf bin/ coverage.out coverage.html

# Tidy dependencies
go mod tidy
```

### mise Tasks

The project uses `mise` for task automation. Available tasks:

```bash
mise run start     # Run the application (go run ./cmd/app)
mise run build     # Build the binary (go build -o accounter ./cmd/app)
mise run test      # Run all tests with coverage
mise run lint      # Run linter
mise run fmt       # Format code
mise run clean     # Clean build artifacts
```

Add more tasks to `mise.toml` as needed:

```toml
[tasks.test]
run = "go test ./... -race -coverprofile=coverage.out && go tool cover -html=coverage.out -o coverage.html"
description = "Run all tests with coverage"

[tasks.lint]
run = "golangci-lint run ./..."
description = "Run linter"

[tasks.fmt]
run = "go fmt ./... && go vet ./..."
description = "Format code"

[tasks.clean]
run = "rm -rf bin/ coverage.out coverage.html accounter"
description = "Clean build artifacts"
```

---

## Architecture & Code Organization

### Folder Structure

```
accounter/
├── cmd/app/main.go              # Entry point: config, DI wiring, server start
├── internal/
│   ├── config/                  # Env var loading, Config struct
│   ├── platform/                # Shared infrastructure
│   │   ├── database/            # SQLite connection
│   │   ├── migrate/             # Migration runner (go:embed)
│   │   ├── auth/                # Bearer token middleware
│   │   ├── logger/              # slog setup
│   │   └── server/              # HTTP server with graceful shutdown
│   ├── category/                # Category domain (model, repo, service, handler)
│   ├── expense/                 # Expense domain (model, repo, service, handler)
│   ├── dashboard/               # Dashboard HTML rendering
│   └── telegram/                # Telegram bot commands
├── web/templates/               # HTML templates (go:embed)
└── docs/PLAN.md                 # Full implementation plan
```

### Dependency Flow

- **Domain packages never import each other** (expense doesn't import category)
- **Repository interfaces defined in domain package** ("accept interfaces, return structs")
- **samber/do used ONLY in cmd/app/main.go** — all other packages use constructor injection
- **All constructors follow pattern**: `func New(dep1, dep2, ...) *Type`

### Layer Responsibilities

| Layer | Responsibility | Examples |
|-------|---------------|----------|
| Handler | HTTP/Telegram I/O, validation, serialization | `handler.go` |
| Service | Business logic, orchestration | `service.go` |
| Repository | Data persistence | `sqlite_repository.go` |
| Model | Domain entities, DTOs | `model.go` |

---

## Code Style Guidelines

### Naming Conventions

**CRITICAL: Use MixedCaps, NEVER underscores (except test subcases, generated code, cgo)**

| Element | Convention | Example |
|---------|-----------|---------|
| Package | lowercase, singular | `expense`, `category` |
| File | lowercase, underscores OK | `expense_handler.go`, `sqlite_repository.go` |
| Exported | UpperCamelCase | `CreateExpense`, `ExpenseService` |
| Unexported | lowerCamelCase | `parseAmount`, `validateInput` |
| Interface | method + `-er` suffix | `Repository`, `CategoryChecker` |
| Error var | `Err` prefix | `ErrNotFound`, `ErrInvalidAmount` |
| Error type | `Error` suffix | `ValidationError` |
| Constructor | `New` or `NewTypeName` | `NewService`, `NewSQLiteRepository` |
| Boolean | `is`, `has`, `can` prefix | `isValid`, `hasPermission` |
| Receiver | 1-2 letter abbreviation | `(s *Service)`, `(r *SQLiteRepository)` |
| Test function | `Test` + name | `TestExpenseService_Create` |
| Acronym | ALL CAPS or all lower | `HTTPServer`, `URL`, `userID` |

### Error Handling

**Golden Rules:**
1. **Errors are either logged OR returned, NEVER both**
2. **Always check errors** — NEVER discard with `_`
3. **Wrap with context**: `fmt.Errorf("fetching user: %w", err)`
4. **Error strings**: lowercase, no trailing punctuation
5. **Use `%w` internally, `%v` at boundaries**
6. **Use `errors.Is`/`errors.As`**, NEVER direct comparison

```go
// Sentinel errors
var ErrNotFound = errors.New("not found")
var ErrInvalidAmount = errors.New("amount must be greater than zero")

// Wrapping with context
if err := svc.Create(ctx, input); err != nil {
    return fmt.Errorf("creating expense: %w", err)
}

// Checking sentinel errors
if errors.Is(err, expense.ErrNotFound) {
    return http.StatusNotFound
}

// Custom error types
type ValidationError struct {
    Field   string
    Message string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("%s: %s", e.Field, e.Message)
}
```

**NEVER use panic for expected errors** — reserve for truly unrecoverable programmer errors.

### Import Organization

```go
import (
    // Standard library first
    "context"
    "fmt"
    "time"
    
    // Third-party packages
    "github.com/go-chi/chi/v5"
    "github.com/samber/do/v2"
    
    // Internal packages
    "github.com/ArtroxGabriel/accounter/internal/config"
    "github.com/ArtroxGabriel/accounter/internal/expense"
)

// Import aliases ONLY on collision
import (
    "crypto/rand"
    mrand "math/rand"
)
```

**NEVER use dot imports** in library code. Blank imports (`_`) only in `main` and test packages.

### Function & Method Patterns

```go
// Early return for errors (keep happy path at minimal indentation)
func (s *Service) Create(ctx context.Context, input CreateInput) (Expense, error) {
    if input.Amount <= 0 {
        return Expense{}, ErrInvalidAmount
    }
    
    exists, err := s.categoryChecker.Exists(ctx, input.CategoryID)
    if err != nil {
        return Expense{}, fmt.Errorf("checking category: %w", err)
    }
    if !exists {
        return Expense{}, ErrCategoryNotFound
    }
    
    return s.repo.Create(ctx, input)
}

// Eliminate unnecessary else
if err != nil {
    return err
}
// happy path continues without else block

// Extract complex conditions
isValidExpense := input.Amount > 0 && input.CategoryID > 0 && !input.Date.IsZero()
if !isValidExpense {
    return ErrInvalidInput
}
```

### Struct & Interface Design

**Interface Principles:**
- Keep interfaces small (1-3 methods)
- **Define interfaces where consumed, not where implemented**
- **Accept interfaces, return structs**
- Don't create interfaces prematurely (wait for 2+ implementations)

```go
// Repository interface defined in domain package
type Repository interface {
    Create(ctx context.Context, input CreateExpenseInput) (Expense, error)
    GetByID(ctx context.Context, id int64) (Expense, error)
    List(ctx context.Context, filter ListFilter) ([]Expense, error)
    Delete(ctx context.Context, id int64) error
    Summary(ctx context.Context, from, to time.Time) (Summary, error)
}

// CategoryChecker: small interface for cross-domain validation
// Defined in expense package, satisfied by category.Service
type CategoryChecker interface {
    Exists(ctx context.Context, id int64) (bool, error)
}
```

**Struct Best Practices:**
- Make zero value useful when possible
- Be consistent with pointer vs value receivers across all methods
- Use struct tags for all exported serialized fields

```go
type Expense struct {
    ID          int64     `json:"id"`
    Amount      int64     `json:"amount"`        // Cents (NOT float)
    Description string    `json:"description"`
    CategoryID  int64     `json:"category_id"`
    Category    string    `json:"category"`       // Denormalized (read-only)
    Date        time.Time `json:"date"`
    CreatedAt   time.Time `json:"created_at"`
}
```

---

## Testing Standards

### Test Structure

**MUST use table-driven tests with named subtests:**

```go
func TestExpenseService_Create(t *testing.T) {
    tests := []struct {
        name        string
        input       CreateExpenseInput
        setupMocks  func(*mockRepository, *mockCategoryChecker)
        wantErr     error
        wantAmount  int64
    }{
        {
            name: "valid input creates expense",
            input: CreateExpenseInput{
                Amount:      2550,
                Description: "Almoço",
                CategoryID:  1,
                Date:        time.Now(),
            },
            setupMocks: func(repo *mockRepository, checker *mockCategoryChecker) {
                checker.exists = true
                repo.createResult = Expense{ID: 42, Amount: 2550}
            },
            wantAmount: 2550,
        },
        {
            name: "zero amount returns error",
            input: CreateExpenseInput{
                Amount: 0,
            },
            wantErr: ErrInvalidAmount,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange
            repo := &mockRepository{}
            checker := &mockCategoryChecker{}
            if tt.setupMocks != nil {
                tt.setupMocks(repo, checker)
            }
            svc := NewService(repo, checker, testLogger())
            
            // Act
            got, err := svc.Create(context.Background(), tt.input)
            
            // Assert
            if tt.wantErr != nil {
                require.ErrorIs(t, err, tt.wantErr)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.wantAmount, got.Amount)
        })
    }
}
```

### Test Organization

- **Co-locate tests**: `expense.go` → `expense_test.go`
- **Independent tests use `t.Parallel()`**
- **Integration tests use build tags**: `//go:build integration`
- **Test public API, not implementation details**
- **Repository tests use real in-memory SQLite**
- **Service tests use mock repositories**
- **Handler tests use mock services**

```go
// Repository test with real SQLite
func setupTestDB(t *testing.T) *sql.DB {
    t.Helper()
    db, err := database.Open(":memory:")
    require.NoError(t, err)
    require.NoError(t, migrate.Run(db))
    t.Cleanup(func() { db.Close() })
    return db
}

func TestSQLiteRepository_Create(t *testing.T) {
    db := setupTestDB(t)
    repo := NewSQLiteRepository(db)
    // test against real SQLite
}
```

### Test Helpers

```go
// Test logger (discards output)
func testLogger() *slog.Logger {
    return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// Helper functions use t.Helper()
func seedCategories(t *testing.T, db *sql.DB) {
    t.Helper()
    _, err := db.Exec("INSERT INTO categories (name, icon) VALUES (?, ?)", "Alimentação", "🍔")
    require.NoError(t, err)
}
```

---

## Domain-Specific Rules

### Money Handling

**CRITICAL: Store all amounts as integer cents, NEVER floats**

```go
// Amount stored as int64 cents
type Expense struct {
    Amount int64 `json:"amount"` // R$ 25.50 → 2550
}

// Parsing user input
amount, err := strconv.ParseFloat(input, 64)
if err != nil {
    return 0, fmt.Errorf("invalid amount: %w", err)
}
cents := int64(amount * 100)
```

### Date Handling

- **Store all dates in UTC** (SQLite uses `datetime('now')`)
- **Convert to timezone only on display** (use `TIMEZONE` env var)
- **Date filter ranges**: `FROM` inclusive, `TO` exclusive

### Authentication

- **Bearer token middleware protects ALL routes except `/health`**
- **Telegram bot uses separate token** (validated by Telegram API)
- **Optional: restrict bot to specific chat ID** via env var

---

## Critical Implementation Notes

1. **`samber/do` DI ONLY in `cmd/app/main.go`** — all other packages use plain constructors
2. **Foreign key constraints enabled**: `PRAGMA foreign_keys=ON;`
3. **WAL mode for SQLite**: `PRAGMA journal_mode=WAL;`
4. **Graceful shutdown**: HTTP server + Telegram bot stop on SIGINT/SIGTERM
5. **Migrations embedded**: Use `//go:embed migrations/*.sql`
6. **Templates embedded**: Use `//go:embed templates/*.html`
7. **Structured logging**: JSON in prod, text in dev (via `slog`)
8. **Low-cardinality error messages**: Don't interpolate variable data into error strings
9. **Context propagation**: Every service/repository method accepts `context.Context`
10. **HTTP status codes**: 201 for Create, 204 for Delete, 404 for NotFound, 400 for validation errors

---

## Common Pitfalls to Avoid

❌ **NEVER** use `panic` for expected errors  
❌ **NEVER** log and return the same error  
❌ **NEVER** use underscores in Go identifiers (except test subcases)  
❌ **NEVER** use floats for money  
❌ **NEVER** import `samber/do` outside of `cmd/app/main.go`  
❌ **NEVER** make domain packages depend on each other  
❌ **NEVER** discard errors with `_`  
❌ **NEVER** write to nil maps (initialize first)  
❌ **NEVER** compare interfaces directly (use `errors.Is`)  
❌ **NEVER** use `init()` for initialization (use explicit constructors)

✅ **ALWAYS** wrap errors with context  
✅ **ALWAYS** validate at boundaries (handlers/bot)  
✅ **ALWAYS** use table-driven tests with named subtests  
✅ **ALWAYS** check if category exists before creating expense  
✅ **ALWAYS** enable foreign keys in SQLite  
✅ **ALWAYS** use MixedCaps naming  

---

## Quick Reference

**Run single test:** `go test -v -race ./path/to/package -run TestName`  
**Run with coverage:** `go test ./... -race -coverprofile=coverage.out`  
**Lint:** `golangci-lint run ./...`  
**Format:** `go fmt ./...`  
**Build:** `go build -v -o bin/accounter ./cmd/app`  

**mise tasks:** `mise run start`, `mise run build`, `mise run test`, `mise run lint`

For full implementation details, see `docs/PLAN.md`.
