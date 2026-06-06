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

The codebase is split into five concerns, connected by a single interface.

### `pkg/core` — shared contract
- `models.go`: domain types `Project`, `Feature`, `Requirement`, `Task` and the `Status` enum.
  - Statuses: `new`, `in_progress`, `done`, `blocked`, `archived`. `AllStatuses()` returns them in order.
  - All four entity types carry `Status`, `BlockedReason string`, `CreatedAt time.Time`, `UpdatedAt time.Time`.
  - Parent types carry child counts (`FeatureCount`, `RequirementCount`, `TaskCount`) populated only in List queries via correlated subqueries.
- `store.go`: the `Store` interface — all CRUD for all four entities. **Every other package depends on this interface, not on a concrete type.**

### `pkg/store/sqlite` — server-side persistence
Implements `core.Store` against a SQLite database (pure-Go `modernc.org/sqlite`, no CGo). `New(dsn)` opens the DB, sets `MaxOpenConns(1)` (SQLite does not support concurrent writes), and calls `migrate()`.

`migrate()` is **not** versioned — it runs `createSchema()` (all `CREATE TABLE IF NOT EXISTS`) and then `loadStatusIDs()`. **To change the schema: update `createSchema` and delete the database file.**

Status values are normalized into a `statuses` table; all entity tables store `status_id INTEGER REFERENCES statuses(id)`. An in-memory `statusIDs map[core.Status]int64` cache is populated at startup via `loadStatusIDs()` and used on every write.

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

### `pkg/export` — Markdown export
`export.Markdown(ctx, store, projectID, w io.Writer)` walks the full project tree and writes structured Markdown: project as `# heading`, features as `##`, requirements as `###`, tasks as `- [ ]` / `- [x]` todo items. Done items are struck through (`~~text~~`). `export.Filename(projectName)` slugifies the name to a safe `.md` filename.

### `pkg/tui` — Bubbletea TUI
`tui.New(store core.Store)` accepts any `Store` implementation. Navigation is hierarchical:
```
Projects → Features → Requirements → Tasks
```

**List screen keys:**
| Key | Action |
|-----|--------|
| `↑`/`↓` or `j`/`k` | move cursor |
| `enter` | drill into selected item |
| `esc` | go back |
| `n` | create new item |
| `e` | edit selected item |
| `d` | delete (asks for confirmation) |
| `x` | export project as Markdown (projects screen only) |
| `q` | quit (asks for confirmation) |
| `ctrl+c` | force quit immediately |

**Form keys:** `tab`/`shift+tab` move between fields; `←`/`→` (or `h`/`l`) cycle status on the status field; a Blocked Reason field appears only when `blocked` is selected; `enter` on the `[ Save ]` button submits; `esc` cancels.

**Directory picker (triggered by `x` on a project):** `↑`/`↓` navigate entries; `enter` opens a subdirectory; `space` exports to the currently displayed directory; `esc` cancels.

After any save or delete, `reloadAll()` fires `tea.Batch` to reload every list in the current navigation path (projects always; features/requirements/tasks when their parent is selected), keeping child counts in sync immediately.

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
3. Implement the methods in `pkg/store/sqlite/sqlite.go` (add a table in `createSchema()`).
4. Add HTTP handler file in `pkg/api/` and register routes in `server.go`.
5. Add client methods to `pkg/client/client.go`.
6. Add list item wrapper and screen/form handling to `pkg/tui/app.go`.
