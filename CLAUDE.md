# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build -o botdo .

# Run (default: :8080, data file: botdo.json)
go run . [--addr :8080] [--data botdo.json]

# Test
go test ./...

# Single package test
go test ./internal/store/...

# Lint (requires golangci-lint)
golangci-lint run
```

## Architecture

**botdo** is a self-hosted task list server with a REST API and embedded web UI, designed to support AI agent workflows (currently Claude Code).

### Data flow

```
main.go → store.JSONStore (botdo.json) ← api.handler → chi router → embedded web/
```

- `main.go`: Parses flags, initializes `JSONStore`, embeds `web/` via `//go:embed`, wires router.
- `internal/model/task.go`: Core `Task` struct with `Status` (`pending`, `in_progress`, `done`, `failed`) and `Agent` (`""`, `"claude-code"`).
- `internal/store/store.go`: `Store` interface + `ErrNotFound` sentinel + `Filter` struct for query params.
- `internal/store/json.go`: `JSONStore` — mutex-protected, atomic file writes via temp file + rename. All reads return copies.
- `internal/api/handler.go`: HTTP handlers. Includes agent-specific endpoints: `claimTask` (pending→in_progress) and `completeTask` (→done/failed).
- `internal/api/router.go`: chi router wiring; serves `/api/*` and falls through to static file server for the web UI.

### API routes

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/tasks` | List tasks (query: `?status=`, `?agent=`) |
| POST | `/api/tasks` | Create task |
| GET | `/api/tasks/{id}` | Get task |
| PUT | `/api/tasks/{id}` | Update task (partial, pointer fields) |
| DELETE | `/api/tasks/{id}` | Delete task |
| POST | `/api/tasks/{id}/claim` | Agent claims task (pending→in_progress) |
| POST | `/api/tasks/{id}/complete` | Agent completes task (body: `{"status":"done"\|"failed"}`) |
| GET | `/api/agents/{agent}/tasks` | Agent polling: pending+in_progress tasks for that agent |

### Adding a new store backend

Implement `store.Store` interface (5 methods), return it from a constructor, and wire it in `main.go`.