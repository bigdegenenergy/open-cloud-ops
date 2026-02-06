#!/bin/bash
# SessionStart Hook - Context Injection
# Provides Claude with repository context at the start of each session
#
# This eliminates the "cold start" problem where Claude doesn't know
# the current state of your codebase. Now every session begins with
# relevant context automatically injected.
#
# Output is sent to stdout and becomes part of Claude's initial context.

echo "## Current Repository State"
echo ""

# ============================================
# GIT STATUS
# ============================================

if git rev-parse --git-dir > /dev/null 2>&1; then
    echo "### Git Status"
    echo '```'
    git status --short --branch 2>/dev/null || echo "Unable to get git status"
    echo '```'
    echo ""

    # Show recent commits for context
    echo "### Recent Commits"
    echo '```'
    git log --oneline -5 2>/dev/null || echo "No commits found"
    echo '```'
    echo ""

    # Show any stashed changes
    STASH_COUNT=$(git stash list 2>/dev/null | wc -l)
    if [ "$STASH_COUNT" -gt 0 ]; then
        echo "### Stashed Changes: $STASH_COUNT"
        echo '```'
        git stash list | head -3
        echo '```'
        echo ""
    fi
fi

# ============================================
# ACTIVE TODOS
# ============================================

echo "### Active TODOs in Codebase"
echo '```'
# Search for TODOs, capturing output once to avoid double I/O
# Excludes common non-source directories for performance
TODO_OUTPUT=$(grep -r "TODO:" \
    --include="*.ts" --include="*.tsx" --include="*.js" --include="*.jsx" \
    --include="*.py" --include="*.go" --include="*.rs" \
    --exclude-dir=node_modules --exclude-dir=.git --exclude-dir=dist \
    --exclude-dir=build --exclude-dir=.next --exclude-dir=__pycache__ \
    --exclude-dir=.venv --exclude-dir=venv --exclude-dir=target \
    . 2>/dev/null)

if [ -n "$TODO_OUTPUT" ]; then
    TODO_COUNT=$(echo "$TODO_OUTPUT" | wc -l)
    echo "$TODO_OUTPUT" | head -10
    if [ "$TODO_COUNT" -gt 10 ]; then
        echo "... and $((TODO_COUNT - 10)) more TODOs"
    fi
else
    echo "No TODOs found"
fi
echo '```'
echo ""

# ============================================
# PROJECT TYPE DETECTION
# ============================================

echo "### Project Configuration"
echo '```'

# Detect project type and show relevant info
if [ -f "package.json" ]; then
    echo "Node.js project detected"
    # Show key scripts if they exist
    if command -v jq &> /dev/null; then
        SCRIPTS=$(jq -r '.scripts | keys[]' package.json 2>/dev/null | head -5 | tr '\n' ', ' | sed 's/,$//')
        if [ -n "$SCRIPTS" ]; then
            echo "  Available scripts: $SCRIPTS"
        fi
    fi
fi

if [ -f "pyproject.toml" ] || [ -f "setup.py" ] || [ -f "requirements.txt" ]; then
    echo "Python project detected"
fi

if [ -f "Cargo.toml" ]; then
    echo "Rust project detected"
fi

if [ -f "go.mod" ]; then
    echo "Go project detected"
fi

if [ -f "Makefile" ]; then
    echo "Makefile found"
fi

if [ -f "docker-compose.yml" ] || [ -f "docker-compose.yaml" ]; then
    echo "Docker Compose configuration found"
fi

echo '```'
echo ""

# ============================================
# FAILING TESTS (if cached)
# ============================================

# Use project-relative path or environment variable for test output
# This avoids security issues with hardcoded global /tmp paths
TEST_LOG="${CLAUDE_TEST_OUTPUT_LOG:-.claude/artifacts/test_output.log}"

if [ -f "$TEST_LOG" ]; then
    # Check if the log is recent (within last hour)
    if [ "$(find "$TEST_LOG" -mmin -60 2>/dev/null)" ]; then
        FAILED_TESTS=$(grep -E "(FAIL|ERROR|failed)" "$TEST_LOG" 2>/dev/null | head -5)
        if [ -n "$FAILED_TESTS" ]; then
            echo "### Recent Test Failures"
            echo '```'
            echo "$FAILED_TESTS"
            echo '```'
            echo ""
        fi
    fi
fi

# ============================================
# ENVIRONMENT
# ============================================

echo "### Environment"
echo '```'
echo "Working directory: $(pwd)"
echo "Branch: $(git branch --show-current 2>/dev/null || echo 'N/A')"
if [ -n "$CLAUDE_STRICT_MODE" ]; then
    echo "Strict mode: ENABLED"
fi
echo '```'
