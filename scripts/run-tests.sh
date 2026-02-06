#!/usr/bin/env bash
# =============================================================================
# Open Cloud Ops - Test Runner
# =============================================================================
# Runs tests for all three application modules and prints a summary.
#
# Usage:
#   ./scripts/run-tests.sh
#   make test
# =============================================================================

set -uo pipefail

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Track results
CEREBRA_RESULT=0
ECONOMIST_RESULT=0
AEGIS_RESULT=0

# ---------------------------------------------------------------------------
# Helper Functions
# ---------------------------------------------------------------------------
header() {
    echo ""
    echo -e "${BLUE}=============================================${NC}"
    echo -e "${BLUE}  $*${NC}"
    echo -e "${BLUE}=============================================${NC}"
    echo ""
}

# ---------------------------------------------------------------------------
# Cerebra Tests (Go)
# ---------------------------------------------------------------------------
header "Running Cerebra (Go) Tests"

if [ -f "${PROJECT_ROOT}/cerebra/go.mod" ]; then
    (cd "${PROJECT_ROOT}/cerebra" && go test -v -race -count=1 ./...) 2>&1
    CEREBRA_RESULT=$?
else
    echo -e "${YELLOW}[SKIP] No go.mod found in cerebra/${NC}"
    CEREBRA_RESULT=-1
fi

# ---------------------------------------------------------------------------
# Aegis Tests (Go)
# ---------------------------------------------------------------------------
header "Running Aegis (Go) Tests"

if [ -f "${PROJECT_ROOT}/aegis/go.mod" ]; then
    (cd "${PROJECT_ROOT}/aegis" && go test -v -race -count=1 ./...) 2>&1
    AEGIS_RESULT=$?
else
    echo -e "${YELLOW}[SKIP] No go.mod found in aegis/${NC}"
    AEGIS_RESULT=-1
fi

# ---------------------------------------------------------------------------
# Economist Tests (Python)
# ---------------------------------------------------------------------------
header "Running Economist (Python) Tests"

if [ -f "${PROJECT_ROOT}/economist/requirements.txt" ]; then
    (cd "${PROJECT_ROOT}/economist" && python3 -m pytest -v --tb=short .) 2>&1
    ECONOMIST_RESULT=$?
else
    echo -e "${YELLOW}[SKIP] No requirements.txt found in economist/${NC}"
    ECONOMIST_RESULT=-1
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
header "Test Summary"

format_result() {
    local name="$1"
    local result="$2"
    if [ "$result" -eq 0 ]; then
        echo -e "  ${GREEN}PASS${NC}  $name"
    elif [ "$result" -eq -1 ]; then
        echo -e "  ${YELLOW}SKIP${NC}  $name"
    else
        echo -e "  ${RED}FAIL${NC}  $name (exit code: $result)"
    fi
}

format_result "Cerebra  (Go)"     "$CEREBRA_RESULT"
format_result "Aegis    (Go)"     "$AEGIS_RESULT"
format_result "Economist (Python)" "$ECONOMIST_RESULT"

echo ""

# Exit with failure if any test suite failed
OVERALL=0
[ "$CEREBRA_RESULT" -gt 0 ] && OVERALL=1
[ "$AEGIS_RESULT" -gt 0 ] && OVERALL=1
[ "$ECONOMIST_RESULT" -gt 0 ] && OVERALL=1

if [ "$OVERALL" -eq 0 ]; then
    echo -e "${GREEN}All test suites passed (or were skipped).${NC}"
else
    echo -e "${RED}One or more test suites failed.${NC}"
fi

echo ""
exit "$OVERALL"
