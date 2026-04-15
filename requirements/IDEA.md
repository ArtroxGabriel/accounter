# Accounter — Personal Expense Tracker

## Goal

Personal app for expense tracking with Telegram integration as one of multiple
input methods, a web dashboard for visualization, and a clean architecture that
allows future extension (investments tab, Google Sheets export, swappable storage).

## Core Features (v1)

- Log expenses via Telegram bot (command-based) or web UI
- Categorize expenses
- Dashboard: filter by day, month, year, custom period
- Simple Bearer token authentication
- Structured logging (slog) across all layers
- Dockerized (single container)

## Tech Stack

| Layer       | Tech                                |
|-------------|-------------------------------------|
| Backend     | Go + chi/v5                         |
| Frontend    | html/template + HTMX + Tailwind CDN |
| Persistence | SQLite (modernc.org/sqlite)         |
| DI          | samber/do                           |
| Telegram    | go-telegram-bot-api/v5              |
| Logging     | log/slog                            |
| Task runner | mise                                |

## Architecture

Layered monolith (single binary, single Docker container):

<ask> recommend architecture(files, project etc) </ask>

```
handler → service → repository → db
```

- Repository interface allows storage layer swap in the future
- samber/do used for DI wiring in main.go only; tests wire manually

## Telegram Commands (v1)

- `/add <amount> <description> <category>` — add expense
- `/list` — last 5 expenses
- `/today` — total spent today
- `/month` — total spent this month
- `/categories` — list available categories

## Development Approach

- TDD: tests written first, manual dependency wiring in tests
- Migrations: custom internal package using go:embed + schema_migrations table
- `.env`: godotenv (non-fatal) in dev; raw env vars in Docker
- Google Sheets export: deferred to post-v1

## Future (post-v1)

- Investment tracking tab (inspired by Nubank, with distribution of investiments by type, performance over time, etc.)
- Google Sheets export
- Swappable persistence layer (e.g., Postgres)
