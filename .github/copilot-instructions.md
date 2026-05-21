# Jirascrap — Repository Overview

> Terminal UI for browsing Jira tickets. Built with Bubble Tea + Lipgloss + SQLite.

## Architecture

```
main.go → config.Load() → jira.NewClient() → store.Open() → logger.SetPersister() → tui.Run()

┌───────────────────────────────────────────────┐
│  TUI (AppModel)                               │
│  ┌─────────────────────────────────────────┐  │
│  │ Views:                                  │  │
│  │  ListModel        — ticket list         │  │
│  │  DetailModel      — ticket detail + md  │  │
│  │  CommentInputModel— add comment + @mention│ │
│  │  TagModel         — tag popup overlay   │  │
│  │  TodoModel        — todo popup overlay  │  │
│  │  StatusModel      — status transition   │  │
│  │  ToastModel       — temp notifications  │  │
│  └─────────────────────────────────────────┘  │
│  PopupManager: visibility, key routing,       │
│    overlay rendering                          │
│  Handlers: split by domain (handlers_*.go)    │
│  Messages: typed results for async operations │
└───────────────────────────────────────────────┘
        │                          │
        ▼                          ▼
  ┌───────────┐           ┌──────────────────┐
  │ Jira API  │           │ SQLite Stores    │
  │ (Client)  │           │  TagStore        │
  │ + HTTP    │           │  TodoStore       │
  │  transport│           │  TicketCache     │
  └───────────┘           │  LogStore        │
                          └──────────────────┘
```

## Package Map

| Package | Path | Purpose |
|---------|------|---------|
| `config` | `internal/config/` | Env-var loader: Domain, Email, APIToken, DBPath, CopilotWorkspace, CopilotModel, AllowHTTP |
| `jira` | `internal/jira/` | HTTP client for Jira REST API v3. Split into 4 sub-interfaces (`TicketFetcher`, `CommentClient`, `UserSearcher`, `TransitionClient`) composed into `TicketClient`. `client.go` (interfaces) + `client_tickets.go` + `client_comments.go` + `client_transitions.go` + `client_users.go` + `http.go` (transport/retry) + `adf_comment.go` (ADF builder with mentions) |
| `model` | `internal/model/` | Domain types: `Ticket`, `Todo`, `Comment`, `User` |
| `store` | `internal/store/` | SQLite persistence via 4 narrow interfaces. Goose migrations. Split into `TagStore`, `TodoStore`, `TicketCache`, `LogStore` |
| `logger` | `internal/logger/` | Thread-safe log buffer (max 100 entries) + DB-backed persistence via `LogPersister` interface. Global `Log` singleton. GooseLoggerAdapter |
| `tmux` | `internal/tmux/` | Wrapper around `tmux` CLI for Copilot integration |
| `tui` | `internal/tui/` | Bubble Tea app: AppModel, PopupManager, handlers (split by domain), messages, copilot launcher |
| `views` | `internal/tui/views/` | Sub-models: ListModel, DetailModel, CommentInputModel, TagModel, TodoModel, StatusModel, ToastModel |
| `keymaps` | `internal/tui/keymaps/` | Central key binding registry (DefaultKeyMap) |

## Handler Files

| File | Responsibility |
|------|---------------|
| `handlers_keys.go` | Key routing: `handleKeyPress`, `globalKeyHandlers` chain, `handleOtherMsg`, comment input guard |
| `handlers_sync.go` | `updateSyncMsg` sub-router, ticket sync/cache, refresh, error handling |
| `handlers_navigation.go` | `updateNavigationMsg` sub-router, ticket selection, epic nav, copilot launch, browser open |
| `handlers_popups.go` | `updatePopupMsg` + `updateStatusMsg` sub-routers, tag/todo/status handlers |
| `handlers_comments.go` | `updateCommentMsg` sub-router, comment fetch/post, user search |

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

### `model.Comment`
```go
type Comment struct {
    ID, Author string
    CreatedAt  time.Time
    Markdown   string
}
```

### `model.User`
```go
type User struct { AccountID, DisplayName string }
```

### `model.Todo`
```go
type Todo struct { Title string; Done bool }
```

### `store` interfaces
- **`TagStore`**: `SaveMeta(id, tags)` / `GetUniqueTags()`
- **`TodoStore`**: `GetTodos(ticketID)` / `SaveTodos(ticketID, todos)`
- **`TicketCache`**: `CacheTickets(tickets)` / `GetCachedTickets()` / `CacheEpicChildren(epicKey, tickets)` / `GetAllCachedEpicChildren()`
- **`LogStore`**: `InsertLog(level, message)` / `GetRecentLogs(limit)`

### `jira` interfaces (composed into `TicketClient`)
- **`TicketFetcher`**: `FetchTickets()` / `FetchEpicChildren(epicKey)` / `FetchAllEpicChildren(tickets)`
- **`CommentClient`**: `FetchComments(issueKey, maxResults)` / `PostComment(issueKey, body)`
- **`UserSearcher`**: `SearchUsers(query)`
- **`TransitionClient`**: `FetchTransitions(issueKey)` / `DoTransition(issueKey, transitionID)`

## Data Flow

1. **Startup**: Load config → create client → open DB (run migrations) → wire log persister → `tui.Run()`
2. **Init**: Spinner + load cached tickets from DB + background sync from Jira API
3. **Sync**: Fetch tickets → cache in SQLite → re-read with tags joined → update UI
4. **Tags**: `t` key → TagModel popup → `SaveMeta()` → reload all views
5. **Todos**: `n` key → TodoModel popup → `SaveTodos()`
6. **Epics**: Select epic → fetch children (or use cache) → show in sub-list. `esc` returns.
7. **Comments**: Enter detail view → lazy fetch last 20 comments → render below ticket body
8. **Add Comment**: `a` key → CommentInputModel textarea → `@` triggers user search autocomplete → enter submits → POST ADF to Jira → refresh comments
9. **Copilot**: `c` key → write markdown prompt to `/tmp/jirascrap/` → tmux window → launch copilot CLI
10. **Status**: `s` key → StatusModel dropdown → fetch transitions from Jira → select → DoTransition → optimistic update + re-sync

## Update() Message Routing

`AppModel.Update()` dispatches messages via sub-routers to keep cyclomatic complexity low:

```
Update(msg) → switch type:
  WindowSizeMsg        → handleWindowSize
  KeyPressMsg          → handleKeyPress (handlers_keys.go)
  sync messages        → updateSyncMsg (handlers_sync.go)
  navigation messages  → updateNavigationMsg (handlers_navigation.go)
  popup messages       → updatePopupMsg (handlers_popups.go)
  status messages      → updateStatusMsg (handlers_popups.go)
  comment messages     → updateCommentMsg (handlers_comments.go)
  ErrMsg              → handleError
  ToastTimeoutMsg     → inline
  default             → handleOtherMsg
```

`handleKeyPress` uses a `keyHandler` chain pattern — global key handlers are iterated via `globalKeyHandlers()` slice. When comment input is active, keys route directly to the active model, bypassing global bindings.

## Database Schema (7 migrations)

1. `issue_tags` — `(id TEXT, tag TEXT)` — ticket tags
2. `issue_todos` — `(id, ticket_id, title, done)` — per-ticket todos
3. `tickets` — `(id, summary, reporter, status, status_category, priority, type, created_at, updated_at, markdown, epic_id)` — cached tickets
4. `logs` — `(id, level, message, created_at)` — application logs

## UI Patterns

- **Overlay composition**: Base view + layers (tag/todo/status/toast) via `lipgloss.NewCompositor`
- **Message passing**: Async commands return typed messages, no blocking
- **Popup routing**: `isPopupActive()` guards global keys; popups get keys first
- **Comment input guard**: `isCommentInputActive()` bypasses all global keys so typing works
- **Epic navigation**: `previousList` stores parent list; `restoreRootList()` pops back
- **ActiveModel interface**: `Update(KeyPressMsg) → (ActiveModel, Cmd)` + `View() → tea.View`
- **MsgUpdater interface**: Optional, for models handling non-key messages (cursor blink etc.)
- **ADF with mentions**: `BuildCommentADF(text, mentions)` converts text + `@name→accountId` map to ADF doc with mention nodes. Sorts mentions longest-first to avoid partial matches.

## Key Bindings

| Key | Action | Key | Action |
|-----|--------|-----|--------|
| `enter` | Select | `t` | Tag |
| `esc` | Back | `n` | Todo |
| `h` | Home | `s` | Status transition |
| `c` | Copilot | `o` | Browser |
| `r` | Refresh | `a` | Add comment |
| `/` | Filter | `?` | Help |
| `q` | Quit | | |

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
- `charm.land/bubbles/v2` — UI components (list, textinput, textarea, viewport, help)
- `charm.land/glamour/v2` — Markdown terminal renderer
- `charm.land/lipgloss/v2` — Terminal styling + layer compositor
- `github.com/mattn/go-sqlite3` — SQLite driver
- `github.com/pressly/goose/v3` — DB migrations

## Standing Rules

- Always update README, mock server (`e2e/mock_server.go`), and e2e tape (`e2e/demo.tape`) when adding new features
- `JIRASCRAP_ALLOW_HTTP=1` env var bypasses HTTPS validation (for e2e/testing with mock server)
- `store.Open(dbPath, gooseLogger)` accepts a `goose.Logger` — pass `logger.GooseLoggerAdapter{}` from `main.go`
- Logging goes to SQLite `logs` table via `LogPersister` interface — no file-based logging
