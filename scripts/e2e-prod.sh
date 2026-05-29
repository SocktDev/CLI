#!/usr/bin/env bash
# Production E2E tests for the Sockt CLI (https://api.sockt.dev).
# Requires: SOCKT_API_KEY (credits), built sockt binary, lnbot MCP or CLI for lightning payments.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$CLI_DIR/.." && pwd)"
CLI="${SOCKT_BIN:-$CLI_DIR/sockt}"
MAIN="${REPO_ROOT}/main.py"
REMOTE="/home/sandbox/main.py"

CREDITS_TIER="${CREDITS_TIER:-nano}"
LIGHTNING_TIER="${LIGHTNING_TIER:-nano}"
LIGHTNING_PREPAID_SATS="${LIGHTNING_PREPAID_SATS:-10}"

RUN_CREDITS=1
RUN_LIGHTNING=1
RUN_NEGATIVE=1

while [[ $# -gt 0 ]]; do
  case "$1" in
    --credits-only) RUN_LIGHTNING=0; RUN_NEGATIVE=0 ;;
    --lightning-only) RUN_CREDITS=0; RUN_NEGATIVE=0 ;;
    --negative-only) RUN_CREDITS=0; RUN_LIGHTNING=0 ;;
    -h|--help)
      echo "Usage: $0 [--credits-only|--lightning-only|--negative-only]"
      echo "Env: SOCKT_API_KEY (required for credits), SOCKT_BIN, CREDITS_TIER, LIGHTNING_TIER, LIGHTNING_PREPAID_SATS"
      exit 0
      ;;
    *) echo "Unknown option: $1" >&2; exit 1 ;;
  esac
  shift
done

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'
PASS=0
FAIL=0

pass() { PASS=$((PASS + 1)); echo -e "${GREEN}PASS${NC}: $1"; }
fail() { FAIL=$((FAIL + 1)); echo -e "${RED}FAIL${NC}: $1"; exit 1; }

SANDBOX_ID=""
cleanup() {
  if [[ -n "${SANDBOX_ID:-}" ]]; then
    "$CLI" sandbox terminate "$SANDBOX_ID" 2>/dev/null || true
  fi
}

wait_pod_ready() {
  # create --wait already waits for data plane; short fallback only
  local id=$1
  for _ in $(seq 1 5); do
    if "$CLI" files ls "$id" 2>/dev/null | grep -q bashrc; then
      return 0
    fi
    sleep 2
  done
  return 1
}

validate_main_py_output() {
  local out=$1
  local lines
  lines=$(printf '%s\n' "$out" | grep -cE '^[0-9]+$' || true)
  [[ "$lines" -eq 10 ]] || fail "main.py: expected 10 lines, got $lines (output: $out)"
  while IFS= read -r n; do
    [[ -z "$n" ]] && continue
    [[ "$n" -ge 1 && "$n" -le 100 ]] || fail "main.py: invalid int $n"
  done <<< "$out"
}

assert_err_contains() {
  local label=$1
  shift
  local out
  set +e
  out=$("$@" 2>&1)
  set -e
  if printf '%s' "$out" | grep -qiE 'unauthorized|not_found|404|terminated'; then
    pass "$label"
  elif printf '%s' "$out" | grep -q '401'; then
    pass "$label"
  else
    fail "$label (got: $out)"
  fi
}

pay_lightning_invoice() {
  local bolt11=$1
  if command -v lnbot >/dev/null 2>&1; then
    lnbot pay --yes "$bolt11" && return 0
  fi
  echo "Pay BOLT11 via lnbot (MCP send_payment or: lnbot pay --yes <invoice>):"
  echo "$bolt11"
  read -r -p "Press Enter after payment, or Ctrl-C to abort..."
}

run_credits_e2e() {
  [[ -n "${SOCKT_API_KEY:-}" ]] || fail "SOCKT_API_KEY is required for credits tests"

  trap cleanup EXIT
  SANDBOX_ID=""

  "$CLI" tiers | grep -q nano && pass "tiers lists nano" || fail "tiers"

  local create_err
  create_err=$(mktemp)
  SANDBOX_ID=$("$CLI" sandbox create --tier "$CREDITS_TIER" --billing credits --label e2e-cli-credits --wait 2>"$create_err")
  rm -f "$create_err"
  [[ -n "$SANDBOX_ID" ]] && pass "credits create ($CREDITS_TIER)" || fail "credits create"

  wait_pod_ready "$SANDBOX_ID" && pass "credits pod ready" || fail "credits pod ready"

  "$CLI" sandbox status "$SANDBOX_ID" | grep -qi running && pass "credits status running" || fail "credits status"
  "$CLI" sandbox status "$SANDBOX_ID" | grep -qi credits && pass "credits billing" || fail "credits billing"

  "$CLI" files write "$SANDBOX_ID" "$REMOTE" --file "$MAIN" 2>&1 | grep -q Written && pass "files write" || fail "files write"
  "$CLI" files ls "$SANDBOX_ID" | grep -q main.py && pass "files ls" || fail "files ls"
  [[ "$("$CLI" files read "$SANDBOX_ID" main.py)" == "$(cat "$MAIN")" ]] && pass "files read" || fail "files read"

  local out
  out=$("$CLI" exec "$SANDBOX_ID" python3 -u "$REMOTE")
  validate_main_py_output "$out"
  pass "exec main.py (python3)"

  "$CLI" exec --workdir /home/sandbox "$SANDBOX_ID" 'python3 -c "import sys; print(sys.version)"' | grep -qE '^[0-9]+\.[0-9]+' && pass "exec python version" || fail "python version"
  [[ "$("$CLI" exec "$SANDBOX_ID" echo hello | tr -d '\n')" == "hello" ]] && pass "exec echo" || fail "exec echo"

  "$CLI" sandbox pause "$SANDBOX_ID" && pass "pause" || fail "pause"
  "$CLI" sandbox status "$SANDBOX_ID" | grep -qi paused && pass "status paused" || fail "status paused"
  "$CLI" sandbox resume "$SANDBOX_ID" && pass "resume" || fail "resume"
  for _ in $(seq 1 30); do
    "$CLI" sandbox status "$SANDBOX_ID" | grep -qi running && break
    sleep 2
  done
  "$CLI" sandbox status "$SANDBOX_ID" | grep -qi running && pass "resumed running" || fail "resumed"

  set +e
  "$CLI" exec "$SANDBOX_ID" "sh -c 'exit 1'"
  local ec=$?
  set -e
  [[ "$ec" -eq 1 ]] && pass "exec exit code 1" || fail "exec nonzero exit=$ec"

  local save=$SANDBOX_ID
  SANDBOX_ID=""
  trap - EXIT
  "$CLI" sandbox terminate "$save" | grep -q terminated && pass "terminate" || fail "terminate"
}

run_lightning_e2e() {
  trap cleanup EXIT
  SANDBOX_ID=""

  local create_err bolt11 token
  create_err=$(mktemp)
  SANDBOX_ID=$(env -u SOCKT_API_KEY "$CLI" sandbox create \
    --tier "$LIGHTNING_TIER" --billing lightning \
    --prepaid-sats "$LIGHTNING_PREPAID_SATS" --label e2e-cli-ln 2>"$create_err")
  token=$(grep 'Sandbox token:' "$create_err" | awk '{print $NF}')
  bolt11=$(grep -o 'lnbc[^[:space:]]*' "$create_err" | head -1)
  rm -f "$create_err"

  [[ -n "$SANDBOX_ID" && -n "$token" && -n "$bolt11" ]] && pass "lightning create" || fail "lightning create parse"
  export SOCKT_SANDBOX_TOKEN="$token"

  pay_lightning_invoice "$bolt11" && pass "lightning invoice paid" || fail "lightning payment"

  for _ in $(seq 1 40); do
    if "$CLI" sandbox status "$SANDBOX_ID" | grep -qi 'Status:[[:space:]]*running'; then
      pass "lightning running"
      break
    fi
    sleep 5
  done
  "$CLI" sandbox status "$SANDBOX_ID" | grep -qi running || fail "lightning not running"

  wait_pod_ready "$SANDBOX_ID" && pass "lightning pod ready" || fail "lightning pod ready"

  "$CLI" files write "$SANDBOX_ID" "$REMOTE" --file "$MAIN" 2>&1 | grep -q Written && pass "lightning files write" || fail "lightning write"
  "$CLI" files ls "$SANDBOX_ID" | grep -q main.py && pass "lightning files ls" || fail "lightning ls"

  local out
  out=$("$CLI" exec "$SANDBOX_ID" python3 -u "$REMOTE")
  validate_main_py_output "$out"
  pass "lightning exec main workload"

  "$CLI" sandbox status "$SANDBOX_ID" | grep -qi lightning && pass "lightning billing status" || fail "lightning status"
  "$CLI" sandbox status "$SANDBOX_ID" | grep -qi 'Time remaining' && pass "lightning time remaining" || fail "lightning balance"

  local save=$SANDBOX_ID
  SANDBOX_ID=""
  trap - EXIT
  env -u SOCKT_API_KEY "$CLI" sandbox terminate "$save" | grep -q terminated && pass "lightning terminate" || fail "lightning terminate"
  unset SOCKT_SANDBOX_TOKEN
}

run_negative_e2e() {
  assert_err_contains "no auth" env -u SOCKT_API_KEY -u SOCKT_SANDBOX_TOKEN \
    "$CLI" sandbox create --tier nano --billing credits

  assert_err_contains "bad api key" env SOCKT_API_KEY=invalid \
    "$CLI" sandbox create --tier nano --billing credits

  export SOCKT_API_KEY="${SOCKT_API_KEY:?SOCKT_API_KEY required}"
  assert_err_contains "unknown sandbox" "$CLI" sandbox status 00000000-0000-0000-0000-000000000000

  local create_err old
  create_err=$(mktemp)
  old=$("$CLI" sandbox create --tier nano --billing credits --wait 2>"$create_err")
  rm -f "$create_err"
  "$CLI" sandbox terminate "$old" >/dev/null
  sleep 3
  assert_err_contains "files on terminated" "$CLI" files write "$old" "$REMOTE" --file "$MAIN"
}

main() {
  [[ -x "$CLI" ]] || fail "sockt binary not found at $CLI (run: cd cli && go build -o sockt .)"

  echo "Sockt CLI production E2E"
  echo "API: ${SOCKT_API_URL:-https://api.sockt.dev}"
  echo "CLI: $CLI"
  echo "Commit: $(git -C "$REPO_ROOT" rev-parse --short HEAD 2>/dev/null || echo unknown)"
  echo "---"

  [[ "$RUN_CREDITS" -eq 1 ]] && run_credits_e2e
  [[ "$RUN_LIGHTNING" -eq 1 ]] && run_lightning_e2e
  [[ "$RUN_NEGATIVE" -eq 1 ]] && run_negative_e2e

  echo "---"
  echo -e "${GREEN}All tests passed${NC}: $PASS checks"
}

main "$@"
