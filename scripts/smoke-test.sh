#!/bin/bash
# smoke-test.sh — Read-only smoke test against a live Dynatrace account.
#
# Runs dtiam commands that only read data (GET operations). Makes ZERO mutations.
# Safe to run against any account at any time.
#
# Required env vars:
#   DTIAM_ACCOUNT_UUID      — Dynatrace account UUID
#   DTIAM_CLIENT_SECRET     — OAuth2 client secret (or use DTIAM_BEARER_TOKEN)
#
# Optional:
#   DTIAM_CLIENT_ID         — OAuth2 client ID (auto-extracted from secret if not set)
#   DTIAM_BEARER_TOKEN      — Static bearer token (alternative to OAuth2)
#   DTIAM_ENVIRONMENT_URL   — Environment URL for apps/schemas tests
#
# Usage:
#   export DTIAM_ACCOUNT_UUID=abc-123
#   export DTIAM_CLIENT_SECRET=dt0s01.XXX.YYY
#   ./scripts/smoke-test.sh

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

PASS=0
FAIL=0
SKIP=0

pass() { echo -e "${GREEN}[PASS]${NC} $1"; ((PASS++)); }
fail() { echo -e "${RED}[FAIL]${NC} $1"; ((FAIL++)); }
skip() { echo -e "${YELLOW}[SKIP]${NC} $1"; ((SKIP++)); }

DTIAM="./bin/dtiam"

# --- Preflight checks ---

echo "dtiam Smoke Test — Read-Only"
echo "============================="
echo ""

if [ ! -f "$DTIAM" ]; then
    echo "Binary not found at $DTIAM — building..."
    make build
fi

if [ -z "${DTIAM_ACCOUNT_UUID:-}" ]; then
    echo -e "${RED}ERROR: DTIAM_ACCOUNT_UUID not set${NC}"
    echo "Set required env vars and try again."
    exit 1
fi

if [ -z "${DTIAM_CLIENT_SECRET:-}" ] && [ -z "${DTIAM_BEARER_TOKEN:-}" ]; then
    echo -e "${RED}ERROR: Neither DTIAM_CLIENT_SECRET nor DTIAM_BEARER_TOKEN set${NC}"
    exit 1
fi

echo "Account: ${DTIAM_ACCOUNT_UUID}"
echo "Auth: $([ -n "${DTIAM_CLIENT_SECRET:-}" ] && echo "OAuth2" || echo "Bearer token")"
echo ""

# --- Helper ---

run_test() {
    local name="$1"
    shift
    if output=$("$@" 2>&1); then
        pass "$name"
        return 0
    else
        fail "$name"
        echo "       $output" | head -3
        return 1
    fi
}

run_json_test() {
    local name="$1"
    shift
    if output=$("$@" --plain -o json 2>&1); then
        # Verify it's valid JSON
        if echo "$output" | python3 -m json.tool > /dev/null 2>&1; then
            pass "$name (JSON valid)"
        else
            fail "$name (invalid JSON)"
        fi
    else
        fail "$name"
    fi
}

# --- Read-only command tests ---

echo "=== Core Resources ==="

run_test "get groups" $DTIAM get groups
run_test "get users" $DTIAM get users
run_test "get policies" $DTIAM get policies
run_test "get bindings" $DTIAM get bindings
run_test "get environments" $DTIAM get environments
run_test "get boundaries" $DTIAM get boundaries

echo ""
echo "=== Account ==="

run_test "account limits" $DTIAM account limits
run_test "account subscriptions" $DTIAM account subscriptions
run_test "account capabilities" $DTIAM account capabilities

echo ""
echo "=== JSON Output ==="

run_json_test "get groups" $DTIAM get groups
run_json_test "get users" $DTIAM get users
run_json_test "account limits" $DTIAM account limits

echo ""
echo "=== Describe (first group) ==="

GROUP_UUID=$($DTIAM get groups --plain -o json 2>/dev/null | python3 -c "import sys,json; data=json.load(sys.stdin); print(data[0]['uuid'] if data else '')" 2>/dev/null || echo "")

if [ -n "$GROUP_UUID" ]; then
    run_test "describe group $GROUP_UUID" $DTIAM describe group "$GROUP_UUID"
    run_test "group members $GROUP_UUID" $DTIAM group members "$GROUP_UUID"
    run_test "group bindings $GROUP_UUID" $DTIAM group bindings "$GROUP_UUID"
else
    skip "describe group (no groups found)"
    skip "group members (no groups found)"
    skip "group bindings (no groups found)"
fi

echo ""
echo "=== Analyze ==="

run_test "analyze least-privilege" $DTIAM analyze least-privilege

echo ""
echo "=== Templates ==="

run_test "template list" $DTIAM template list
run_test "template show policy-readonly" $DTIAM template show policy-readonly
run_test "template render group-team" $DTIAM template render group-team --set name=SmokeTest --set description=test
run_test "template path" $DTIAM template path

echo ""
echo "=== Environment Resources ==="

if [ -n "${DTIAM_ENVIRONMENT_URL:-}" ]; then
    run_test "get apps --environment" $DTIAM get apps --environment "${DTIAM_ENVIRONMENT_URL}"
    run_test "get schemas --environment" $DTIAM get schemas --environment "${DTIAM_ENVIRONMENT_URL}"
else
    skip "get apps (DTIAM_ENVIRONMENT_URL not set)"
    skip "get schemas (DTIAM_ENVIRONMENT_URL not set)"
fi

echo ""
echo "=== Dry-Run Safety Check ==="

run_test "create group --dry-run" $DTIAM create group --name "SMOKE-TEST-DO-NOT-CREATE" --dry-run
run_test "delete group --dry-run" $DTIAM delete group "SMOKE-TEST-DO-NOT-DELETE" --dry-run
run_test "apply --dry-run" $DTIAM template apply group-team --set name=SMOKE-TEST --dry-run

# --- Summary ---

echo ""
echo "============================="
echo -e "${GREEN}Passed:${NC}  $PASS"
echo -e "${YELLOW}Skipped:${NC} $SKIP"
echo -e "${RED}Failed:${NC}  $FAIL"
echo ""

if [ $FAIL -gt 0 ]; then
    echo -e "${RED}Smoke test FAILED${NC}"
    exit 1
else
    echo -e "${GREEN}Smoke test PASSED${NC}"
    exit 0
fi
