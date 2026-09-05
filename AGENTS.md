# Jirascrap — Repository Overview

> Terminal UI for browsing Jira tickets. Built with Bubble Tea + Lipgloss + SQLite.

## Definition of Done — read this first

**No development task is complete until `mise run check` passes.** Building and eyeballing
the diff is not sufficient.

After finishing any code change, run:

```bash
mise run check
```

That is the local gate. It runs, in order: `gofmt` → `go vet` → `golangci-lint` →
`sqlc diff` → `go test -race`.

Static analysis is **SonarQube Cloud, running in GitHub Actions** — there is no local
SonarQube. The `sonar` job scans every push to `main` and every pull request and fails the
build when the quality gate is red (`-Dsonar.qualitygate.wait=true`).

Rules:

- Do **not** report work as finished before `mise run check` has passed.
- `mise run check` is necessary but not sufficient. The Sonar gate only exists in CI, so
  once a change is pushed, the `SonarQube Cloud` job is the authoritative verdict. Say
  which of the two you actually ran when you summarise.
- If the quality gate fails, fix the reported conditions. Do not rationalise a red gate
  away, and do not weaken `sonar-project.properties` exclusions to turn it green.
- Run `mise run sonar-issues` to read the current findings. The SonarQube Cloud project is
  private, so this needs `SONAR_TOKEN` set to a Cloud **user** token with Browse permission
  (an `sqp_` analysis token will not work). The task fails loudly if it cannot read the
  project — an empty result is never reported as "no issues".
- Do not reintroduce a local scanner. Scanning an uncommitted working tree from a laptop
  publishes it as a main-branch analysis and corrupts the new-code baseline.

## Architecture

```
main.go → config.Load() → jira.NewClient() → store.Open() → logger.Log.SetPersister() → tui.Run()

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
  └───────────┘           │  SqliteLogStore  │
                          └──────────────────┘
```

## Package Map

| Package | Path | Purpose |
|---------|------|---------|
| `config` | `internal/config/` | Env-var loader + validation. Fields: `Domain`, `Email`, `APIToken`, `DBPath` |
| `jira` | `internal/jira/` | HTTP client for Jira REST API v3. Split into 4 sub-interfaces (`TicketFetcher`, `CommentClient`, `UserSearcher`, `TransitionClient`) composed into `TicketClient`. `client.go` (interfaces + `Client`) + `client_tickets.go` + `client_comments.go` + `client_transitions.go` + `client_users.go` + `http.go` (transport/retry) + `model.go` (API wire types) + `adf_comment.go` (ADF builder with mentions) + `adfmd.go` (ADF → Markdown) |
| `model` | `internal/model/` | Domain types: `Ticket`, `Todo`, `Comment`, `User` |
| `store` | `internal/store/` | SQLite persistence. 3 narrow interfaces (`TagStore`, `TodoStore`, `TicketCache`) + concrete `SqliteLogStore`. Goose migrations, sqlc-generated queries |
| `sqlcdb` | `internal/store/sqlcdb/` | **Generated** by sqlc. Do not edit by hand |
| `logger` | `internal/logger/` | Thread-safe log buffer (max 100 entries) + DB persistence via `LogPersister` interface. Global `Log` singleton |
| `tui` | `internal/tui/` | Bubble Tea app: `AppModel` (`app.go`), `PopupManager` (`popups.go`), messages (`messages.go`), handlers split by domain |
| `views` | `internal/tui/views/` | Sub-models: ListModel, DetailModel, CommentInputModel, TagModel, TodoModel, StatusModel, ToastModel + shared `types.go` / `styles.go` / `comments.go` |
| `keymaps` | `internal/tui/keymaps/` | Central key binding registry (`DefaultKeyMap` in `keymaps.go`) |

## Handler Files

| File | Responsibility |
|------|---------------|
| `handlers_keys.go` | Key routing: `handleKeyPress`, `globalKeyHandlers` chain, `isCommentInputActive` guard, `handleOtherMsg` |
| `handlers_sync.go` | `updateSyncMsg` sub-router, ticket sync/cache, refresh, error handling |
| `handlers_navigation.go` | `updateNavigationMsg` sub-router, `handleWindowSize`, ticket selection, epic nav, browser open |
| `handlers_popups.go` | `updatePopupMsg` + `updateStatusMsg` sub-routers, tag/todo/status handlers |
| `handlers_comments.go` | `updateCommentMsg` sub-router, comment fetch/post, user search. `maxComments = 20` |

## Key Types

### `model.Ticket`
```go
type Ticket struct {
    ID, Summary, Reporter, Status, StatusCategory string
    CreatedAt, UpdatedAt time.Time
    Markdown string
    Tags     []string
    Priority string
    Type     string  // "Epic", "Task", "Story", "Bug"
    EpicID   *string // nil for top-level tickets
}

func (t Ticket) IsEpic() bool // Type == "Epic"
```

`EpicID` is a **pointer** — `nil` means top-level. This is load-bearing: `GetCachedTickets`
filters on `epic_id IS NULL`, and `CacheTickets` only clears top-level rows.

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
type Todo struct {
    ID    int // database ID (0 for unsaved)
    Title string
    Done  bool
}
```

### `store` interfaces
- **`TagStore`**: `SaveTags(id, tags)` / `GetUniqueTags()`
- **`TodoStore`**: `GetTodos(ticketID)` / `SaveTodos(ticketID, todos)`
- **`TicketCache`**: `CacheTickets(tickets)` / `GetCachedTickets()` / `CacheEpicChildren(epicKey, tickets)` / `GetAllCachedEpicChildren()`
- **`SqliteLogStore`** (struct, not an interface): `InsertLog(level, message)` — satisfies `logger.LogPersister`

All writes go through `withTx(db, func(q *sqlcdb.Queries) error)` in `helpers.go`.

### `jira` interfaces (composed into `TicketClient`)
All methods take `ctx context.Context` as the first argument.

- **`TicketFetcher`**: `FetchTickets(ctx)` / `FetchEpicChildren(ctx, epicKey)` / `FetchAllEpicChildren(ctx, tickets)`
- **`CommentClient`**: `FetchComments(ctx, issueKey, maxResults) ([]model.Comment, int, error)` / `PostComment(ctx, issueKey, body any)`
- **`UserSearcher`**: `SearchUsers(ctx, query)`
- **`TransitionClient`**: `FetchTransitions(ctx, issueKey) ([]Transition, error)` / `DoTransition(ctx, issueKey, transitionID)`

Pagination is not implemented — the client assumes fewer than `maxResults` (100) issues per query.

## Data Flow

1. **Startup**: Load config → create client → open DB (goose migrations run here) → wire log persister → `tui.Run()`
2. **Init**: Spinner + load cached tickets from DB + background sync from Jira API
3. **Sync**: Fetch tickets → cache in SQLite → re-read with tags joined → update UI
4. **Tags**: `t` → TagModel popup → `SaveTags()` → reload all views
5. **Todos**: `n` → TodoModel popup → `SaveTodos()`
6. **Epics**: Select epic → fetch children (or use cache) → show in sub-list. `esc` returns.
7. **Comments**: Enter detail view → lazy fetch last 20 comments → render below ticket body
8. **Add Comment**: `a` → CommentInputModel textarea → `@` triggers user search autocomplete → enter submits → POST ADF to Jira → refresh comments
9. **Status**: `s` → StatusModel dropdown → fetch transitions from Jira → select → DoTransition → optimistic update + re-sync

## Update() Message Routing

`AppModel.Update()` dispatches messages via sub-routers to keep cyclomatic complexity low:

```
Update(msg) → switch type:
  tea.WindowSizeMsg    → handleWindowSize            (handlers_navigation.go)
  tea.KeyPressMsg      → handleKeyPress              (handlers_keys.go)

  cachedTicketsLoadedMsg | syncCompleteMsg | syncErrorMsg
                       → updateSyncMsg               (handlers_sync.go)

  views.SelectTicketMsg | views.GoToListMsg
  | epicChildrenLoadedMsg | epicChildrenErrorMsg
                       → updateNavigationMsg         (handlers_navigation.go)

  views.TagsFilledMsg | views.TodosChangedMsg
  | tagSavedMsg | todoSavedMsg
                       → updatePopupMsg              (handlers_popups.go)

  transitionsLoadedMsg | transitionsErrorMsg
  | views.StatusTransitionMsg
  | statusTransitionCompleteMsg | statusTransitionErrorMsg
                       → updateStatusMsg             (handlers_popups.go)

  commentsLoadedMsg | commentsErrorMsg
  | views.CommentSubmitMsg | views.CommentCancelMsg
  | commentPostSuccessMsg | commentPostErrorMsg
  | views.UserSearchRequestMsg | userSearchResultMsg
                       → updateCommentMsg            (handlers_comments.go)

  views.ErrMsg         → handleError
  views.ToastTimeoutMsg→ inline
  default              → handleOtherMsg
```

`handleKeyPress` uses a `keyHandler` chain — global handlers are iterated via the
`globalKeyHandlers()` slice. When comment input is active, keys route directly to the
active model, bypassing all global bindings.

**When adding a new async operation:** define the message in `messages.go`, add it to the
matching `case` group above, and handle it in that domain's sub-router. Do not add a
top-level `case` to `Update()` — that is what the sub-routers exist to prevent.

## Database Schema (7 goose migrations)

1. `issue_tags` — `(id, tag)` — ticket tags
2. `issue_todos` — `(id, ticket_id, title, done)` — per-ticket todos
3. `tickets` — `(id, summary, reporter, status, status_category, priority, type, created_at, updated_at, markdown, epic_id)` — cached tickets
4. `logs` — `(id, level, message, created_at)` — application logs

`tickets` and `logs` are disposable cache. `issue_tags` and `issue_todos` hold real user
data and are deliberately **not** foreign-keyed to `tickets`, so they survive a sync that
drops a ticket from the result set.

### Code generation

`internal/store/migrations/` is the single source of truth. Goose applies the migrations at
runtime; sqlc reads the same directory directly (it understands `-- +goose Up` and ignores
Down) to generate `internal/store/sqlcdb/` from the queries in `internal/store/queries/`.

```
migrations/*.sql ─┬─ goose ──> runtime DB
                  └─ sqlc  ──> internal/store/sqlcdb/  (committed)
```

`internal/store/sqlcdb/` **is committed**, so a fresh clone builds with no codegen step.
sqlc is a mise tool (version pinned in `mise.toml`), not a Go `tool` directive — that keeps
its ~50 transitive deps (pgx, mysql, tidb parser, grpc, cel-go, wazero) out of `go.mod`.

## UI Patterns

- **Overlay composition**: base view + layers (tag/todo/status/toast) via `lipgloss.NewCompositor`
- **Message passing**: async commands return typed messages, never block
- **Popup routing**: `PopupManager.IsActive()` guards global keys; popups get keys first via `RouteKeyPress`
- **Comment input guard**: `isCommentInputActive()` bypasses all global keys so typing works
- **Epic navigation**: `navLevel == navEpic` + `previousList` holds the parent list; `rootList()` returns the real root either way
- **`ActiveModel` interface**: `Update(tea.KeyPressMsg) (ActiveModel, tea.Cmd)` + `View() tea.View`
- **`MsgUpdater` interface**: optional, for models handling non-key messages (cursor blink etc.)
- **`popupState` embed**: gives `Hide()` / `IsVisible()` to popup structs for free
- **`baseDelegate` embed**: gives `Height()` / `Spacing()` / `Update()` to list delegates; only `Render()` needs implementing
- **ADF with mentions**: `BuildCommentADF(text, mentions)` converts text + `@name→accountId` map to an ADF doc with mention nodes. Sorts mentions longest-first to avoid partial matches

## Key Bindings

Defined in `internal/tui/keymaps/keymaps.go`. `keymaps_test.go` asserts no two global
bindings share a key — add new bindings to that test's list.

| Key | Action | Key | Action |
|-----|--------|-----|--------|
| `enter` | Select | `t` | Tag |
| `esc` | Back | `n` | Todo |
| `H` | Home (capital) | `s` | Status transition |
| `r` | Refresh | `o` | Open in browser |
| `?` | Toggle help | `a` | Add comment |
| `q` | Quit | `ctrl+c` | Force quit |
| `/` | Filter (from bubbles list) | | |

Tag popup: `tab` autocomplete, `up`/`down` navigate, `enter` save, `esc` cancel.
Todo popup: `a` add, `space`/`enter` toggle, `x` delete, `esc` close.

## Build & Test

```bash
go build -o jirascrap.out .   # or: mise run build
go test ./...                  # all tests
go run .                       # dev run
mise run generate              # regen sqlc code (only after migration/query changes)
mise run generate-check        # verify committed sqlc code is current (sqlc diff)
mise run coverage              # tests + coverage.out (consumed by Sonar in CI)
mise run sonar-issues          # list open SonarQube Cloud findings (needs SONAR_TOKEN)
mise run check                 # the full local gate — run this before declaring work done
bash e2e/run.sh                # e2e demo (needs vhs, ttyd, ffmpeg)
```

## CI

`.github/workflows/ci.yml`, three jobs:

- `test` — checkout → setup Go → gofmt check → build → vet → test (`-race`) → coverage artifact → gosec
- `codegen` — checkout → setup sqlc (version read from `mise.toml`) → `sqlc diff` to catch stale committed generated code
- `sonar` — checkout (`fetch-depth: 0`) → setup Go → test with coverage → `SonarSource/sonarqube-scan-action` with `-Dsonar.qualitygate.wait=true`

The `sonar` job needs the `SONAR_TOKEN` repository secret. No `SONAR_HOST_URL` is set —
the scan action defaults to SonarQube Cloud. Project and organization keys live in
`sonar-project.properties`.

**Automatic Analysis must stay off** (project → Administration → Analysis Method). Sonar
refuses to run both analysis methods: with Automatic Analysis enabled, the CI-based
analysis fails and takes the build down with it. Automatic Analysis also ignores
`sonar-project.properties` entirely and cannot ingest coverage, so it would silently drop
the `**/sqlcdb/**` exclusion and all coverage data. CI-based analysis is the deliberate
choice here for exactly those two reasons.

## Dependencies

- `charm.land/bubbletea/v2` — TUI framework
- `charm.land/bubbles/v2` — UI components (list, textinput, textarea, viewport, help)
- `charm.land/glamour/v2` — Markdown terminal renderer
- `charm.land/lipgloss/v2` — Terminal styling + layer compositor
- `modernc.org/sqlite` — pure-Go SQLite driver (driver name: `sqlite`, **not** mattn/go-sqlite3)
- `github.com/pressly/goose/v3` — DB migrations

## Standing Rules

- **Run `mise run check` before declaring any task done — see "Definition of Done" above**
- Always update README, mock server (`e2e/mock_server.go`), and e2e tape (`e2e/demo.tape`) when adding new features
- After editing `internal/store/migrations/` or `internal/store/queries/`, run `mise run generate` and **commit** the regenerated `internal/store/sqlcdb/` — CI fails otherwise
- Never add sqlc back as a Go `tool` directive; it belongs in `mise.toml`
- Never enable SonarQube Cloud's Automatic Analysis — it conflicts with the CI `sonar` job and fails the build
- Never hand-edit `internal/store/sqlcdb/` — it is excluded from golangci-lint and Sonar
- Migrations are append-only: add a new numbered file, never edit an applied one
- New global key bindings must be added to the conflict test in `keymaps_test.go`
- `JIRASCRAP_ALLOW_HTTP=1` bypasses the HTTPS check in `config.Validate()` (for e2e/testing against the mock server)
- `store.Open(dbPath, gooseLogger)` takes a `goose.Logger` — `main.go` passes its local `gooseLogger{}` adapter
- Logging goes to the SQLite `logs` table via the `LogPersister` interface — no file-based logging
