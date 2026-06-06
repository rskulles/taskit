# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Fetch / tidy dependencies
go mod tidy

# Build everything
go build ./...

# Build specific binaries
go build -o bin/taskitd ./cmd/taskitd
go build -o bin/taskit  ./cmd/taskit

# Run the server (default port 42069, db at ~/.local/share/taskit/taskit.db)
go run ./cmd/taskitd
go run ./cmd/taskitd -addr :9000 -db ./dev.db   # custom overrides

# Run the TUI client
go run ./cmd/taskit
go run ./cmd/taskit -server http://localhost:42069

# Run tests
go test ./...
go test ./pkg/store/sqlite/...   # single package
```

## Architecture

The codebase is split into four concerns, connected by a single interface.

### `pkg/core` — shared contract
- `models.go`: domain types `Project`, `Feature`, `Requirement`, `Task` and the `Status` enum (`active`, `in_progress`, `done`, `blocked`, `archived`).
- `store.go`: the `Store` interface — all CRUD for all four entities. **Every other package depends on this interface, not on a concrete type.**

### `pkg/store/sqlite` — server-side persistence
Implements `core.Store` against a SQLite database (pure-Go `modernc.org/sqlite`, no CGo). `New(dsn)` opens the DB and runs migrations. `SetMaxOpenConns(1)` is intentional — SQLite does not support concurrent writes.

### `pkg/api` — HTTP layer
`api.NewServer(store core.Store)` wraps any `Store` in an `http.Handler`. Routes follow the hierarchy:
```
/projects
/projects/{projectID}/features
/features/{featureID}/requirements
/requirements/{requirementID}/tasks
/projects/{id}   /features/{id}   /requirements/{id}   /tasks/{id}
```
Uses Go 1.22+ method+path pattern matching (`GET /foo/{id}`).

### `pkg/client` — HTTP client
`client.New(baseURL)` returns a `*Client` that implements `core.Store` by calling the API server. The TUI binary uses this.

### `pkg/tui` — Bubbletea TUI
`tui.New(store core.Store)` accepts any `Store` implementation. Navigation is hierarchical:
```
Projects → Features → Requirements → Tasks
```
Keys: `↑/↓` or `j/k` to move, `enter` to drill in, `esc` to go back, `n` to create, `e` to edit, `d` to delete, `q` to quit. Forms use `tab`/`shift+tab` to move between fields; `enter` on the status field saves.

### `cmd/taskitd` — server binary
Wires `sqlite.Store` → `api.Server` → `http.ListenAndServe`.

### `cmd/taskit` — TUI client binary
Wires `client.Client` → `tui.Model` → `tea.Program`.

## Data hierarchy
```
Project
  └─ Feature
       └─ Requirement
            └─ Task
```
All four types carry `Status`. Deleting a parent cascades to children (SQLite `ON DELETE CASCADE`).

## Adding a new entity
1. Add the type to `pkg/core/models.go`.
2. Add CRUD methods to the `Store` interface in `pkg/core/store.go`.
3. Implement the methods in `pkg/store/sqlite/sqlite.go` (add a table in `migrate()`).
4. Add HTTP handler file in `pkg/api/` and register routes in `server.go`.
5. Add client methods to `pkg/client/client.go`.
6. Add list item wrapper and screen/form handling to `pkg/tui/app.go`.
