#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
MOCK_PORT=18932
DB_PATH=$(mktemp -t jirascrap-e2e-XXXXXX.db)

cleanup() {
    if [ -n "${MOCK_PID:-}" ]; then
        kill "$MOCK_PID" 2>/dev/null || true
    fi
    rm -f "$DB_PATH" "$SCRIPT_DIR/mock-server" "$SCRIPT_DIR/jirascrap-e2e" "$SCRIPT_DIR/jirascrap"
}
trap cleanup EXIT

# Build the app and mock server
cd "$ROOT_DIR"
go build -o "$SCRIPT_DIR/jirascrap-e2e" .
go build -o "$SCRIPT_DIR/mock-server" "$SCRIPT_DIR/mock_server.go"

# Symlink so the tape shows a clean command
ln -sf "$SCRIPT_DIR/jirascrap-e2e" "$SCRIPT_DIR/jirascrap"

# Start mock Jira server
PORT=$MOCK_PORT "$SCRIPT_DIR/mock-server" &
MOCK_PID=$!
sleep 1

# Export env vars for the app
export JIRA_BASE_URL="http://localhost:$MOCK_PORT"
export JIRA_EMAIL="test@e2e.com"
export JIRA_API_TOKEN="fake-token"
export JIRA_DB_PATH="$DB_PATH"
export JIRASCRAP_ALLOW_HTTP="1"
export PATH="$SCRIPT_DIR:$PATH"

# Run VHS tape
vhs "$SCRIPT_DIR/demo.tape"
