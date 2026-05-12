# Jirascrap — Repository Overview

> Terminal UI for browsing Jira tickets. Built with Bubble Tea + Lipgloss + SQLite.

## Architecture

```
main.go → config.Load() → jira.NewClient() → logger.OpenSessionLog() → store.Open() → tui.Run()

┌───────────────────────────────────────────────┐
│  TUI (AppModel)                               │
│  ┌─────────────────────────────────────────┐  │
│  │ Views:                                  │  │
│  │  ListModel   — ticket list (bubbles)    │  │
│  │  DetailModel — ticket detail + markdown │  │
│  │  TagModel    — tag popup overlay        │  │
│  │  TodoModel   — todo popup overlay       │  │
│  │  StatusModel — status transition popup  │  │
│  │  ToastModel  — temporary notifications  │  │
│  └─────────────────────────────────────────┘  │
│  PopupManager: visibility, key routing,       │
│    overlay rendering                          │
│  Handlers: key routing, async commands        │
│  Messages: typed results for async operations │
└───────────────────────────────────────────────┘
        │                          │
        ▼                          ▼
  ┌───────────┐           ┌──────────────────┐
  │ Jira API  │           │ SQLite Stores    │
  │ (Client)  │           │  TagStore        │
  │ + HTTP    │           │  TodoStore       │
  │  transport│           │  TicketCache     │
  └───────────┘           └──────────────────┘
```

## Package Map

| Package | Path | Purpose |
|---------|------|---------|
| `config` | `internal/config/` | Env-var loader: Domain, Email, APIToken, DBPath, LogDir, CopilotWorkspace, CopilotModel |
| `jira` | `internal/jira/` | HTTP client for Jira REST API v3. Fetches tickets, epic children. ADF↔Markdown converter. `client.go` (API ops) + `http.go` (transport/retry) |
| `model` | `internal/model/` | Domain types: `Ticket`, `Todo` |
| `store` | `internal/store/` | SQLite persistence via 3 narrow interfaces. Goose migrations. Split into `TagStore`, `TodoStore`, `TicketCache` |
| `logger` | `internal/logger/` | Thread-safe log buffer (max 100 entries) + file-based session logging. Global `Log` singleton. GooseLoggerAdapter |
| `tmux` | `internal/tmux/` | Wrapper around `tmux` CLI for Copilot integration |
| `tui` | `internal/tui/` | Bubble Tea app: AppModel, PopupManager, handlers, messages, copilot launcher |
| `views` | `internal/tui/views/` | Sub-models: ListModel, DetailModel, TagModel, TodoModel, StatusModel, ToastModel |
| `keymaps` | `internal/tui/keymaps/` | Central key binding registry (DefaultKeyMap) |

## Key Types

### `model.Ticket`
```go
type Ticket struct {
    ID, Summary, Reporter, Status, StatusCategory string
    CreatedAt, UpdatedAt time.Time
    Markdown string
    Tags []string
    Priority, Type, EpicID string
}
```

### `model.Todo`
```go
type Todo struct { Title string; Done bool }
```

### `store` interfaces
- **`TagStore`**: `SaveMeta(id, tags)` / `GetUniqueTags()`
- **`TodoStore`**: `GetTodos(ticketID)` / `SaveTodos(ticketID, todos)`
- **`TicketCache`**: `CacheTickets(tickets)` / `GetCachedTickets()` / `CacheEpicChildren(epicKey, tickets)` / `GetAllCachedEpicChildren()`

### `jira.Client`
- `FetchTickets()` — JQL: `assignee = currentUser() AND statusCategory != Done`
- `FetchEpicChildren(epicKey)` — JQL: `"Epic Link" = X OR parent = X`
- `FetchAllEpicChildren(tickets)` — concurrent (max 5 goroutines)
- `FetchTransitions(issueKey)` — GET `/rest/api/3/issue/{key}/transitions`
- `DoTransition(issueKey, transitionID)` — POST `/rest/api/3/issue/{key}/transitions`
- Uses POST `/rest/api/3/search/jql` with basic auth

## Data Flow

1. **Startup**: Load config → create client → open session log → open DB (run migrations) → `tui.Run()`
2. **Init**: Spinner + load cached tickets from DB + background sync from Jira API
3. **Sync**: Fetch tickets → cache in SQLite → re-read with tags joined → update UI
4. **Tags**: `t` key → TagModel popup → `SaveMeta()` → reload all views
5. **Todos**: `n` key → TodoModel popup → `SaveTodos()`
6. **Epics**: Select epic → fetch children (or use cache) → show in sub-list. `esc` returns.
7. **Copilot**: `c` key → write markdown prompt to `/tmp/jirascrap/` → tmux window → launch copilot CLI
8. **Status**: `s` key → StatusModel dropdown → fetch transitions from Jira → select → DoTransition → optimistic update + re-sync

## Database Schema (4 migrations)

1. `issue_tags` — `(id TEXT, tag TEXT)` — ticket tags
2. `issue_todos` — `(id, ticket_id, title, done)` — per-ticket todos
3. `tickets` — `(id, summary, reporter, status, status_category, priority, type, created_at, updated_at, markdown, epic_id)` — cached tickets

## UI Patterns

- **Overlay composition**: Base view + layers (tag/todo/status/toast) via `lipgloss.NewCompositor`
- **Message passing**: Async commands return typed messages, no blocking
- **Popup routing**: `isPopupActive()` guards global keys; popups get keys first
- **Epic navigation**: `previousList` stores parent list; `restoreRootList()` pops back
- **ActiveModel interface**: `Update(KeyPressMsg) → (ActiveModel, Cmd)` + `View() → tea.View`
- **MsgUpdater interface**: Optional, for models handling non-key messages (cursor blink etc.)

## Key Bindings

| Key | Action | Key | Action |
|-----|--------|-----|--------|
| `enter` | Select | `t` | Tag |
| `esc` | Back | `n` | Todo |
| `h` | Home | `s` | Status transition |
| `c` | Copilot | `o` | Browser |
| `r` | Refresh | `?` | Help |
| `/` | Filter | `q` | Quit |

## Build & Test

```bash
go build -o jirascrap.out .   # or: mise run build
go test ./...                  # all tests
go run .                       # dev run
bash e2e/run.sh                # e2e demo (needs vhs, ttyd, ffmpeg)
```

## CI

`.github/workflows/ci.yml` — push to main + PRs: checkout → setup Go → build → vet → test

## Dependencies

- `charm.land/bubbletea/v2` — TUI framework
- `charm.land/bubbles/v2` — UI components (list, textinput, viewport, help)
- `charm.land/glamour/v2` — Markdown terminal renderer
- `charm.land/lipgloss/v2` — Terminal styling + layer compositor
- `github.com/mattn/go-sqlite3` — SQLite driver
- `github.com/pressly/goose/v3` — DB migrations
