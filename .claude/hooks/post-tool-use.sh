#!/bin/bash
# PostToolUse Hook - Runs after every tool use by Claude
# Primary use: Automatic code formatting and linting

# This hook is called with the following environment variables:
# - CLAUDE_TOOL_NAME: The name of the tool that was just used
# - CLAUDE_TOOL_OUTPUT: The output of the tool
# - CLAUDE_SESSION_ID: The current session ID

# Only run formatting if a file was modified
if [[ "$CLAUDE_TOOL_NAME" == "Edit" ]] || [[ "$CLAUDE_TOOL_NAME" == "Write" ]]; then

    # Ensure the reference file exists; create it on first run so -newer works.
    if [[ ! -f /tmp/claude_last_run ]]; then
        touch -t 197001010000 /tmp/claude_last_run
    fi

    echo "🔧 Running post-tool-use formatting..."

    # Python files - Black formatter
    if find . -name "*.py" -newer /tmp/claude_last_run 2>/dev/null | grep -q .; then
        echo "  Formatting Python files with Black..."
        black --quiet . 2>/dev/null || true
        ruff check --fix . 2>/dev/null || true
    fi
    
    # JavaScript/TypeScript files - Prettier
    if find . \( -name "*.js" -o -name "*.ts" -o -name "*.jsx" -o -name "*.tsx" \) -newer /tmp/claude_last_run 2>/dev/null | grep -q .; then
        echo "  Formatting JS/TS files with Prettier..."
        npx prettier --write "**/*.{js,ts,jsx,tsx}" 2>/dev/null || true
        npx eslint --fix . 2>/dev/null || true
    fi
    
    # Go files - gofmt
    if find . -name "*.go" -newer /tmp/claude_last_run 2>/dev/null | grep -q .; then
        echo "  Formatting Go files with gofmt..."
        gofmt -w . 2>/dev/null || true
    fi
    
    # Rust files - rustfmt
    if find . -name "*.rs" -newer /tmp/claude_last_run 2>/dev/null | grep -q .; then
        echo "  Formatting Rust files with rustfmt..."
        cargo fmt 2>/dev/null || true
    fi
    
    # Update timestamp
    touch /tmp/claude_last_run
    
    echo "✅ Formatting complete"
fi

# Track tool usage for metrics (optional)
echo "$(date -Iseconds),${CLAUDE_TOOL_NAME},${CLAUDE_SESSION_ID}" >> .claude/metrics/tool_usage.csv 2>/dev/null || true

# Exit with 0 to continue Claude's execution
exit 0
