#!/usr/bin/env bash
#
# SonarQube helper.
#
# Subcommands:
#   scan    run the scanner (coverage.out must already exist)
#   gate    block until the last analysis finishes, then check the quality gate
#   issues  list unresolved issues
#
# Exit codes:
#   0  success, or Sonar not configured / unreachable (soft skip)
#   1  quality gate failed, analysis failed, or Sonar is misconfigured
#
# Soft-skip vs hard-fail: an unreachable server is treated as "not available right
# now" and skipped with a warning. A reachable server that rejects the token is a
# misconfiguration and fails loudly -- a gate that silently no-ops is worthless.

set -uo pipefail

readonly REPORT=".scannerwork/report-task.txt"
readonly POLL_ATTEMPTS=60
readonly POLL_INTERVAL=2
readonly STARTUP_ATTEMPTS=12
readonly STARTUP_INTERVAL=5

warn() { echo "sonar: $*" >&2; }
die() {
  echo "sonar: $*" >&2
  exit 1
}

# api GET <path-or-url> -> body on stdout, non-zero on HTTP error
api() {
  local url=$1
  case "$url" in
  http*) ;;
  *) url="${SONAR_HOST_URL%/}/${url#/}" ;;
  esac
  curl -sf --max-time 30 -u "${SONAR_TOKEN}:" "$url"
}

# preflight returns:
#   0  configured, reachable, authenticated
#   2  not configured or unreachable (caller should soft-skip)
#   1  reachable but token rejected (caller should hard-fail)
preflight() {
  if [ -z "${SONAR_HOST_URL:-}" ] || [ -z "${SONAR_TOKEN:-}" ]; then
    warn "SONAR_HOST_URL / SONAR_TOKEN not set - skipping."
    return 2
  fi

  # /api/system/status answers 200 with status=STARTING while the server boots,
  # so check the reported status rather than just the HTTP code.
  local sys status i
  status=""
  for ((i = 0; i < STARTUP_ATTEMPTS; i++)); do
    sys=$(curl -sf --max-time 10 "${SONAR_HOST_URL%/}/api/system/status" 2>/dev/null) || sys=""
    status=$(printf '%s' "$sys" | jq -r '.status // empty' 2>/dev/null)
    [ "$status" = "UP" ] && break
    [ -z "$status" ] && break
    warn "server is $status, waiting..."
    sleep "$STARTUP_INTERVAL"
  done

  if [ -z "$status" ]; then
    warn "unreachable at ${SONAR_HOST_URL} - skipping."
    warn "start SonarQube and re-run to enforce the gate."
    return 2
  fi

  if [ "$status" != "UP" ]; then
    warn "server still $status after $((STARTUP_ATTEMPTS * STARTUP_INTERVAL))s - skipping."
    return 2
  fi

  local valid
  valid=$(api "/api/authentication/validate" | jq -r '.valid // false' 2>/dev/null)
  if [ "$valid" != "true" ]; then
    warn "server is up but SONAR_TOKEN was rejected."
    warn "generate a new token at ${SONAR_HOST_URL%/}/account/security"
    warn "and set SONAR_TOKEN in mise.local.toml."
    return 1
  fi

  return 0
}

cmd_scan() {
  preflight
  case $? in
  2) return 0 ;;
  1) return 1 ;;
  esac

  [ -f coverage.out ] || die "coverage.out missing - run 'mise run coverage' first."

  sonar-scanner || die "scanner failed."
}

cmd_gate() {
  preflight
  case $? in
  2) return 0 ;;
  1) return 1 ;;
  esac

  [ -f "$REPORT" ] || die "$REPORT not found - run 'mise run sonar' first."

  local ce_task_url dashboard
  ce_task_url=$(sed -n 's/^ceTaskUrl=//p' "$REPORT")
  dashboard=$(sed -n 's/^dashboardUrl=//p' "$REPORT")
  [ -n "$ce_task_url" ] || die "$REPORT has no ceTaskUrl - re-run 'mise run sonar'."

  # The scanner returns before SonarQube finishes processing, so poll the
  # compute-engine task. A missing task means the report file is stale.
  local task_json status i
  status=""
  for ((i = 0; i < POLL_ATTEMPTS; i++)); do
    if ! task_json=$(api "$ce_task_url"); then
      die "analysis task not found (stale $REPORT) - re-run 'mise run sonar'."
    fi
    status=$(printf '%s' "$task_json" | jq -r '.task.status // empty')
    case "$status" in
    SUCCESS | FAILED | CANCELED) break ;;
    esac
    sleep "$POLL_INTERVAL"
  done

  case "$status" in
  SUCCESS) ;;
  "") die "timed out after $((POLL_ATTEMPTS * POLL_INTERVAL))s waiting for analysis." ;;
  *) die "analysis finished with status $status. See $dashboard" ;;
  esac

  local analysis_id gate gate_status
  analysis_id=$(printf '%s' "$task_json" | jq -r '.task.analysisId // empty')
  [ -n "$analysis_id" ] || die "analysis id missing from compute-engine response."

  gate=$(api "/api/qualitygates/project_status?analysisId=${analysis_id}") ||
    die "could not read quality gate."
  gate_status=$(printf '%s' "$gate" | jq -r '.projectStatus.status // empty')

  if [ "$gate_status" = "OK" ]; then
    echo "sonar: quality gate PASSED"
    return 0
  fi

  warn "quality gate ${gate_status:-UNKNOWN}"
  printf '%s' "$gate" |
    jq -r '.projectStatus.conditions[]?
           | select(.status != "OK")
           | "  \(.metricKey): actual=\(.actualValue) threshold=\(.errorThreshold)"' >&2
  warn "dashboard: $dashboard"
  return 1
}

cmd_issues() {
  preflight
  case $? in
  2) return 0 ;;
  1) return 1 ;;
  esac

  local issues
  issues=$(api "/api/issues/search?projects=${SONAR_PROJECT:-}&resolved=false&ps=500&s=SEVERITY&asc=false") ||
    die "could not list issues."

  echo "# ${SONAR_PROJECT:-project}: $(printf '%s' "$issues" | jq -r '.total') open issue(s)"
  printf '%s' "$issues" |
    jq -r '.issues[]
           | "\(.severity)\t\(.rule)\t\(.component | sub("^[^:]+:";"")):\(.line // 0)\t\(.message)"' |
    column -t -s "$(printf '\t')"
}

main() {
  for tool in curl jq; do
    command -v "$tool" >/dev/null 2>&1 || die "$tool is required but not installed."
  done

  case "${1:-}" in
  scan) cmd_scan ;;
  gate) cmd_gate ;;
  issues) cmd_issues ;;
  *) die "usage: $0 {scan|gate|issues}" ;;
  esac
}

main "$@"
