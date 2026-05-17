# Spec: Render Comments in Detail View

## Problem

The detail view shows ticket metadata + description but no comments. Users need to see Jira issue comments to understand context without switching to browser.

## Approach

Fetch comments lazily (on entering detail view), convert ADF bodies to markdown, render below description. No caching — always fresh. Show last 20, support loading more later. Design comment model to support future write operations (POST new comment).

## Acceptance Criteria

1. When user opens a ticket detail view, comments are fetched from Jira and displayed below description
2. A spinner/loading indicator shows while comments load
3. Each comment shows: author name, timestamp, and rendered markdown body
4. Comments are separated visually (horizontal rule or similar)
5. Maximum 20 most recent comments shown initially
6. If fetching fails, detail view still shows description with an error note for comments section
7. Existing detail view behavior (scrolling, tags, keybinds) is unaffected

## Non-Goals (Future)

- Adding new comments (write API) — design for it, don't implement
- Caching comments in SQLite
- Editing/deleting comments
- Reactions/emoji on comments

## Technical Approach

### 1. Domain Model (`internal/model/comment.go`)

```go
type Comment struct {
    ID        string
    Author    string
    CreatedAt time.Time
    Markdown  string // ADF converted to markdown
}
```

### 2. Jira Client (`internal/jira/client.go`)

Add to `TicketClient` interface:
```go
FetchComments(ctx context.Context, issueKey string, maxResults int) ([]model.Comment, int, error)
// Returns: comments (newest last), total count, error
```

API endpoint: `GET /rest/api/3/issue/{key}/comment?orderBy=-created&maxResults=20`

Response model (`internal/jira/model.go`):
```go
type commentsResponse struct {
    Total    int              `json:"total"`
    Comments []commentEntry   `json:"comments"`
}

type commentEntry struct {
    ID      string        `json:"id"`
    Author  commentAuthor `json:"author"`
    Created jiraTime      `json:"created"`
    Body    any           `json:"body"` // ADF node
}

type commentAuthor struct {
    DisplayName string `json:"displayName"`
}
```

### 3. TUI Messages (`internal/tui/messages.go`)

```go
type commentsLoadedMsg struct {
    ticketID string
    comments []model.Comment
    total    int
}

type commentsErrorMsg struct {
    ticketID string
    err      error
}
```

### 4. Detail View Changes (`internal/tui/views/detail_model.go`)

- Add fields: `comments []model.Comment`, `commentsTotal int`, `commentsLoading bool`, `commentsError error`
- New method: `SetComments(comments, total)` — stores and re-renders viewport
- New method: `SetCommentsError(err)` — stores error and re-renders
- Rendering: After description, add "Comments (N)" header, then each comment block

### 5. Navigation Handler (`internal/tui/handlers_navigation.go`)

- In `handleSelectTicket`: after creating DetailModel, return a `tea.Cmd` that fetches comments
- New handler for `commentsLoadedMsg` / `commentsErrorMsg`: calls `SetComments` or `SetCommentsError` on active DetailModel

### 6. Comment Rendering Format

```
───────────────────────────────
💬 Comments (5 of 12)
───────────────────────────────

**John Smith** · 2024-03-15 14:30

<rendered markdown body>

---

**Jane Doe** · 2024-03-14 09:15

<rendered markdown body>
```

## File Plan

| File | Action | Description |
|------|--------|-------------|
| `internal/model/comment.go` | Create | Comment struct |
| `internal/jira/model.go` | Edit | Add comment response types |
| `internal/jira/client.go` | Edit | Add FetchComments method + interface |
| `internal/tui/messages.go` | Edit | Add comment messages |
| `internal/tui/views/detail_model.go` | Edit | Add comment rendering, loading state |
| `internal/tui/handlers_navigation.go` | Edit | Trigger comment fetch on detail open |
| `internal/tui/app.go` | Edit | Handle comment messages in Update |

## Dependencies

None — uses existing ADF→Markdown converter, existing HTTP transport with retry.
