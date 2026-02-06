#!/bin/bash
# Stop Hook - Runs at the end of Claude's turn
# Primary use: Automated verification and quality gates
#
# This implements the "feedback loop" pattern - Claude verifies its own work.
# As Boris Cherny notes: "Giving Claude a way to verify its work can 2-3x
# the quality of the final result."
#
# Environment variables:
# - CLAUDE_SESSION_ID: The current session ID
# - CLAUDE_TURN_COUNT: The number of turns in this session
# - CLAUDE_STRICT_MODE: Set to "1" to block completion on test failures
#
# Exit codes:
# - 0: All checks passed
# - 1: Some checks failed (Claude is notified but can continue)
# - 2: Critical failure (blocks Claude from declaring task complete)

echo "🔍 Running end-of-turn quality checks..."
echo "   (Feedback loop: verifying Claude's work)"

# Initialize exit code
EXIT_CODE=0
CRITICAL_FAILURE=0

# Check 1: Run tests if they exist
if [ -f "package.json" ] && grep -q "\"test\"" package.json; then
    echo "  Running npm tests..."
    if npm test 2>&1 | tee /tmp/claude_test_output.log; then
        echo "  ✅ Tests passed"
    else
        echo "  ❌ Tests failed"
        EXIT_CODE=1
    fi
elif [ -f "pytest.ini" ] || [ -d "tests" ]; then
    echo "  Running pytest..."
    if pytest --quiet 2>&1 | tee /tmp/claude_test_output.log; then
        echo "  ✅ Tests passed"
    else
        echo "  ❌ Tests failed"
        EXIT_CODE=1
    fi
elif [ -f "Cargo.toml" ]; then
    echo "  Running cargo test..."
    if cargo test --quiet 2>&1 | tee /tmp/claude_test_output.log; then
        echo "  ✅ Tests passed"
    else
        echo "  ❌ Tests failed"
        EXIT_CODE=1
    fi
fi

# Check 2: Type checking
if [ -f "tsconfig.json" ]; then
    echo "  Running TypeScript type checking..."
    if npx tsc --noEmit 2>&1 | tee /tmp/claude_typecheck_output.log; then
        echo "  ✅ Type checking passed"
    else
        echo "  ⚠️  Type checking found issues"
        # Don't fail on type errors, just warn
    fi
elif command -v mypy &> /dev/null && [ -f "pyproject.toml" ]; then
    echo "  Running mypy type checking..."
    if mypy . 2>&1 | tee /tmp/claude_typecheck_output.log; then
        echo "  ✅ Type checking passed"
    else
        echo "  ⚠️  Type checking found issues"
    fi
fi

# Check 3: Linting
if [ -f ".eslintrc.js" ] || [ -f ".eslintrc.json" ]; then
    echo "  Running ESLint..."
    if npx eslint . 2>&1 | tee /tmp/claude_lint_output.log; then
        echo "  ✅ Linting passed"
    else
        echo "  ⚠️  Linting found issues"
    fi
elif command -v ruff &> /dev/null; then
    echo "  Running ruff..."
    if ruff check . 2>&1 | tee /tmp/claude_lint_output.log; then
        echo "  ✅ Linting passed"
    else
        echo "  ⚠️  Linting found issues"
    fi
fi

# Check 4: Security scanning (if tools are available)
if command -v bandit &> /dev/null && find . -name "*.py" | grep -q .; then
    echo "  Running security scan with bandit..."
    if bandit -r . -ll 2>&1 | tee /tmp/claude_security_output.log; then
        echo "  ✅ No security issues found"
    else
        echo "  ⚠️  Security scan found potential issues"
    fi
fi

# Check 5: Check for uncommitted changes
if git diff --quiet && git diff --cached --quiet; then
    echo "  ✅ No uncommitted changes"
else
    echo "  ℹ️  There are uncommitted changes"
fi

# Log metrics
mkdir -p .claude/metrics
echo "$(date -Iseconds),${CLAUDE_SESSION_ID},${CLAUDE_TURN_COUNT},${EXIT_CODE}" >> .claude/metrics/quality_checks.csv 2>/dev/null || true

# Determine final exit code
if [ $EXIT_CODE -eq 0 ]; then
    echo ""
    echo "✅ All quality checks passed"
    echo "   Claude's work has been verified."
    exit 0
else
    echo ""
    echo "❌ Some quality checks failed - review the output above"
    echo ""

    # In strict mode, block Claude from completing if tests failed
    if [ "${CLAUDE_STRICT_MODE:-0}" = "1" ]; then
        echo "⛔ STRICT MODE: Task cannot be marked complete until tests pass."
        echo "   Fix the failing tests and try again."
        exit 2  # Exit code 2 blocks the agent
    else
        echo "ℹ️  Claude has been notified of the failures."
        echo "   Set CLAUDE_STRICT_MODE=1 to block completion on failures."
        exit 1  # Exit code 1 notifies but allows continuation
    fi
fi
