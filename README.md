# jirascrap

A terminal UI for browsing your Jira tickets. Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

Tickets are fetched from the Jira API and cached locally in SQLite. You can tag tickets, manage per-ticket todo lists, filter, and search -- all from the terminal.

![demo](e2e/demo.gif)

## Requirements

- Go 1.22+
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
|----------|----------|-------------|
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
|-----|--------|
| `enter` | Select ticket |
| `esc` | Go back / close popup |
| `t` | Tag current ticket |
| `n` | Open todo list |
| `r` | Refresh tickets from Jira |
| `d` | Toggle debug overlay |
| `?` | Toggle full help |
| `q` | Quit |
| `ctrl+c` | Force quit |
| `/` | Filter tickets |

In the tag popup:

| Key | Action |
|-----|--------|
| `tab` | Autocomplete tag |
| `up/down` | Navigate suggestions |
| `enter` | Save tags |
| `esc` | Cancel |

In the todo popup:

| Key | Action |
|-----|--------|
| `a` | Add new todo |
| `space` | Toggle done |
| `x` | Delete todo |
| `esc` | Close |

## How it works

On startup, jirascrap loads any cached tickets from the local SQLite database and displays them immediately. In the background, it fetches fresh data from the Jira API, updates the cache, and refreshes the UI. This means the app is usable instantly, even on slow connections.

Tags and todos are stored locally and are preserved even if a ticket is removed from your Jira query results.

Press `r` at any time to manually sync with Jira.

## Development

Run the app:

```
go run .
```

Run tests:

```
go test ./...
```

Run the e2e demo (requires [vhs](https://github.com/charmbracelet/vhs), ttyd, and ffmpeg):

```
bash e2e/run.sh
```
