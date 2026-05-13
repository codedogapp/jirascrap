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
| `JIRASCRAP_LOG_DIR` | No | Directory for session log files (default: `~/.local/state/jirascrap/logs/`) |
| `JIRASCRAP_COPILOT_WORKSPACE` | No | Directory where Copilot CLI launches (default: current working directory) |
| `JIRASCRAP_COPILOT_MODEL` | No | AI model for initial planning (default: `claude-haiku-4.5`) |

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
| `s` | Change ticket status |
| `r` | Refresh tickets from Jira |
| `o` | Open ticket in browser |
| `c` | Send ticket to Copilot CLI (tmux) |
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

## Logging

Session logs are written to `~/.local/state/jirascrap/logs/` (configurable via `JIRASCRAP_LOG_DIR`). Each app session creates a timestamped log file. The 10 most recent log files are kept; older ones are pruned automatically.

## Copilot CLI Integration

Press `c` on any ticket to launch [GitHub Copilot CLI](https://docs.github.com/copilot/concepts/agents/about-copilot-cli) in a tmux session with full ticket context.

**How it works:**
- A single tmux session named `copilot` is created (or reused)
- Each ticket gets its own pane, identified by ticket ID
- Copilot starts in plan mode with a cheap model and the ticket context as the initial prompt
- The TUI stays running — switch to the copilot session with `tmux attach -t copilot`

**Requirements:** `tmux` and `copilot` must be in your PATH.

**Configuration:**

| Variable | Default | Description |
|----------|---------|-------------|
| `JIRASCRAP_COPILOT_WORKSPACE` | Current directory | Working directory for Copilot |
| `JIRASCRAP_COPILOT_MODEL` | `claude-haiku-4.5` | Model for initial planning |

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
