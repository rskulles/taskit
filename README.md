# taskit

> **Disclaimer:** This project was vibe coded. Use at your own risk.

This is free and unencumbered software released into the public domain. See [UNLICENSE](UNLICENSE) for details.

A terminal-based task tracker with a client-server architecture. Work is organized as a four-level hierarchy:

```
Project → Feature → Requirement → Task
```

The server stores data in SQLite and exposes a REST API. The client is a Bubbletea TUI that talks to the server over HTTP.

## Requirements

- Go 1.23+
- No CGo — uses `modernc.org/sqlite` (pure Go)

## Getting started

**Build both binaries:**

```bash
go build -o bin/taskitd ./cmd/taskitd
go build -o bin/taskit  ./cmd/taskit
```

**Start the server** (defaults to port `42069`, database at `~/.local/share/taskit/taskit.db`):

```bash
./bin/taskitd
```

**Launch the TUI client:**

```bash
./bin/taskit
```

Both binaries accept flags for custom addresses and paths:

```bash
./bin/taskitd -addr :9000 -db ./dev.db
./bin/taskit  -server http://localhost:9000
```

## TUI controls

### List screens

| Key | Action |
|-----|--------|
| `↑` / `↓` or `j` / `k` | Move cursor |
| `enter` | Open / drill into selected item |
| `esc` | Go back |
| `n` | Create new item |
| `e` | Edit selected item |
| `d` | Delete selected item (with confirmation) |
| `x` | Export project to Markdown (projects screen only) |
| `q` | Quit (with confirmation) |
| `ctrl+c` | Force quit |
| `/` | Filter list |

### Forms

| Key | Action |
|-----|--------|
| `tab` / `shift+tab` | Move between fields |
| `←` / `→` | Cycle through statuses |
| `enter` | Save (when focused on `[ Save ]`) |
| `esc` | Cancel |

A **Blocked reason** field appears automatically when the `blocked` status is selected.

### Markdown export (directory picker)

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate entries |
| `enter` | Open directory |
| `space` | Export to current directory |
| `esc` | Cancel |

## Statuses

All items share the same set of statuses:

| Status | Meaning |
|--------|---------|
| `new` | Not yet started |
| `in_progress` | Actively being worked on |
| `done` | Complete |
| `blocked` | Waiting on something (attach a reason) |
| `archived` | No longer relevant |

## Markdown export

Pressing `x` on a project opens a directory picker. After selecting a destination with `space`, a `.md` file is written there. The filename is derived from the project name (e.g. `my-project.md`).

The export format:
- Project → `# heading`
- Features → `## heading`, separated by `---`
- Requirements → `### heading`
- Tasks → `- [ ] task` / `- [x] ~~done task~~`
- Done items at every level are struck through

## Architecture

| Package | Role |
|---------|------|
| `pkg/core` | Shared domain types (`Project`, `Feature`, `Requirement`, `Task`) and the `Store` interface |
| `pkg/store/sqlite` | SQLite implementation of `Store` |
| `pkg/api` | HTTP server wrapping any `Store` |
| `pkg/client` | HTTP client implementing `Store` |
| `pkg/export` | Markdown export logic |
| `pkg/tui` | Bubbletea TUI, accepts any `Store` |
| `cmd/taskitd` | Server binary |
| `cmd/taskit` | TUI client binary |

The `core.Store` interface is the single seam in the system — the TUI depends only on it, so the SQLite store and HTTP client are interchangeable.
