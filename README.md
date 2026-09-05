# jirascrap

A terminal UI for browsing your Jira tickets. Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

Tickets are fetched from the Jira API and cached locally in SQLite. You can tag tickets, manage per-ticket todo lists, filter, and search -- all from the terminal.

![demo](e2e/demo.gif)

## Requirements

- Go 1.27+
- A Jira Cloud instance with API access

## Installation

```
git clone https://github.com/codedogapp/jirascrap.git
cd jirascrap
go build -o jirascrap .
```

## Configuration

Set these environment variables before running:

| Variable | Required | Description |
| ---------- | ---------- | ------------- |
| `JIRA_BASE_URL` | Yes | Your Jira instance URL (e.g. `https://yourorg.atlassian.net`) |
| `JIRA_EMAIL` | Yes | Email associated with your Atlassian account |
| `JIRA_API_TOKEN` | Yes | API token from [Atlassian API tokens](https://id.atlassian.com/manage-profile/security/api-tokens) |
| `JIRA_DB_PATH` | No | Path to SQLite database file (default: `./data/jira.db`) |

## Usage

```
jirascrap
```

### Key bindings

| Key | Action |
| ----- | -------- |
| `enter` | Select ticket |
| `esc` | Go back / close popup |
| `H` | Return to the root ticket list |
| `t` | Tag current ticket |
| `n` | Open todo list |
| `a` | Add comment (in detail view) |
| `s` | Change ticket status |
| `r` | Refresh tickets from Jira |
| `o` | Open ticket in browser |
| `?` | Toggle full help |
| `q` | Quit |
| `ctrl+c` | Force quit |
| `/` | Filter tickets |

In the tag popup:

| Key | Action |
| ----- | -------- |
| `tab` | Autocomplete tag |
| `up/down` | Navigate suggestions |
| `enter` | Save tags |
| `esc` | Cancel |

In the todo popup:

| Key | Action |
| ----- | -------- |
| `a` | Add new todo |
| `space` | Toggle done |
| `x` | Delete todo |
| `esc` | Close |

## How it works

On startup, jirascrap loads any cached tickets from the local SQLite database and displays them immediately. In the background, it fetches fresh data from the Jira API, updates the cache, and refreshes the UI. This means the app is usable instantly, even on slow connections.

When you open a ticket's detail view, comments are fetched from Jira and rendered below the description. Each comment shows the author, timestamp, and full rendered body. Up to 20 most recent comments are shown.

Tags and todos are stored locally and are preserved even if a ticket is removed from your Jira query results.

Press `r` at any time to manually sync with Jira.

## Logging

Logs are persisted to the same SQLite database used for tickets and tags (in the `logs` table). No separate log files or directories are needed. The in-memory log buffer (max 100 entries) is available for debugging during the session.

## Development

Run the app:

```
go run .
```

Run tests:

```
go test ./...
```

### Database and code generation

The SQLite schema is defined by goose migrations in `internal/store/migrations/`. These
migrations are the single source of truth: goose applies them at runtime, and
[sqlc](https://sqlc.dev) reads them directly to generate database access code from the
queries in `internal/store/queries/`.

The generated package `internal/store/sqlcdb/` is committed, so `go build ./...` and
`go test ./...` work on a fresh clone with no code generation step and no extra tooling.

Regenerate only after changing a migration or a query file:

```
mise run generate
```

Commit the resulting diff. To check for drift without writing files:

```
mise run generate-check
```

sqlc is installed as a mise tool rather than a Go tool dependency, which keeps its large
transitive dependency tree out of `go.mod`. The version is pinned in `mise.toml`; CI reads
that same value, so local and CI codegen always agree. CI verifies the committed output is
up to date in a separate job.

### Code quality gate

Run this before considering a change complete:

```
mise run check
```

It chains `gofmt`, `go vet`, `golangci-lint`, `sqlc diff` and the test suite with `-race`.
It needs no network and no external services.

Static analysis is handled by SonarQube Cloud from GitHub Actions, not locally. The `sonar`
CI job scans each push to `main` and each pull request and fails the build if the quality
gate is red. It requires a `SONAR_TOKEN` repository secret; the project and organization
keys are in `sonar-project.properties`.

SonarQube Cloud's **Automatic Analysis must be turned off** for this project (Administration
→ Analysis Method). It cannot run alongside CI-based analysis, and it ignores
`sonar-project.properties` and coverage reports.

Run the e2e demo (requires [vhs](https://github.com/charmbracelet/vhs), ttyd, and ffmpeg):

```
bash e2e/run.sh
```
