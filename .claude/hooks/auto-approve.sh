#!/bin/bash
# PermissionRequest Hook - Auto-Approve Trusted Commands
# Eliminates approval friction for commands you already trust.
#
# This hook intercepts permission requests and automatically approves
# safe, well-known commands. No more clicking "approve" for pytest.
#
# Output: JSON with decision field
#   {"decision": "approve"} - Auto-approve the command
#   {"decision": "deny", "message": "reason"} - Block the command
#   (no output) - Fall through to normal permission dialog
#
# Exit codes:
#   0 = Hook ran successfully (output determines action)
#   non-zero = Hook failed, fall through to normal behavior
#
# SECURITY NOTES:
# ===============
# DO NOT auto-approve commands that execute user-defined scripts from
# mutable config files (package.json, Makefile, etc.). An agent could:
# 1. Modify package.json to add malicious code to "test" script
# 2. Run "npm test" which would be auto-approved
# 3. Execute arbitrary code without human review
#
# Therefore, we only auto-approve:
# - Direct binary runners (pytest, cargo test, go test)
# - Read-only tools (git status, ls, grep)
# - Formatters that only modify files in expected ways
#
# We DO NOT auto-approve:
# - npm/yarn/pnpm run <anything> (executes package.json scripts)
# - make (executes Makefile targets)
# - Any command that delegates to user-defined config

# Read the permission request from stdin
INPUT=$(cat)

# Extract tool name and details
TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // empty')
BASH_COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

# ============================================
# SECURITY: Check for command chaining
# ============================================

# Function to check if command contains shell metacharacters that could chain commands
# This prevents attacks like "pytest; rm -rf /" or "pytest && malicious"
contains_shell_metacharacters() {
    local cmd="$1"

    # Define forbidden patterns in variables for clarity and safety
    # Pattern 1: Command chaining operators (;, &, |)
    local CHAIN_CHARS='[;&|]'
    # Pattern 2: Command substitution (backticks or $())
    local CMD_SUBST='(`|\$\()'
    # Pattern 3: Output redirection (>)
    local REDIRECT='>'
    # Pattern 4: Newlines (critical - "pytest\nrm -rf /" bypass)
    local NEWLINES=$'[\r\n]'

    if [[ "$cmd" =~ $CHAIN_CHARS ]] || \
       [[ "$cmd" =~ $CMD_SUBST ]] || \
       [[ "$cmd" =~ $REDIRECT ]] || \
       [[ "$cmd" =~ $NEWLINES ]]; then
        return 0  # true - contains dangerous chars
    fi
    return 1  # false - safe
}

# ============================================
# TRUSTED BASH COMMANDS - AUTO APPROVE
# ============================================
#
# SECURITY NOTE: These patterns use prefix matching (^command).
# This means "pytest --some-flag" will also be approved.
# This is intentional to allow legitimate flags like --watch, --coverage.
# The shell metacharacter check above prevents dangerous chaining.
#
# IMPORTANT: We only auto-approve commands that run binaries directly,
# NOT package manager scripts (npm run, yarn, make) which execute
# user-defined code from mutable config files.

if [[ "$TOOL_NAME" == "Bash" ]] && [[ -n "$BASH_COMMAND" ]]; then
    # SECURITY: Never auto-approve commands with shell metacharacters
    if contains_shell_metacharacters "$BASH_COMMAND"; then
        # Fall through to permission dialog for safety
        exit 0
    fi

    # Test commands - ONLY direct binary runners
    # SECURITY: Do NOT auto-approve npm test, yarn test, pnpm test, make test
    # because package.json scripts can be modified to run arbitrary commands
    if [[ "$BASH_COMMAND" =~ ^pytest ]] || \
       [[ "$BASH_COMMAND" =~ ^python\ -m\ pytest ]] || \
       [[ "$BASH_COMMAND" =~ ^cargo\ test ]] || \
       [[ "$BASH_COMMAND" =~ ^go\ test ]]; then
        echo '{"decision": "approve"}'
        exit 0
    fi

    # Lint commands - ONLY direct binary runners
    # SECURITY: Do NOT auto-approve npm run lint, pnpm lint, yarn lint
    # because package.json scripts can be modified to run arbitrary commands
    if [[ "$BASH_COMMAND" =~ ^npx\ eslint ]] || \
       [[ "$BASH_COMMAND" =~ ^ruff\ check ]] || \
       [[ "$BASH_COMMAND" =~ ^flake8 ]] || \
       [[ "$BASH_COMMAND" =~ ^cargo\ clippy ]] || \
       [[ "$BASH_COMMAND" =~ ^golint ]] || \
       [[ "$BASH_COMMAND" =~ ^staticcheck ]] || \
       [[ "$BASH_COMMAND" =~ ^shellcheck ]]; then
        echo '{"decision": "approve"}'
        exit 0
    fi

    # Format commands - safe, modifies files but in expected ways
    if [[ "$BASH_COMMAND" =~ ^npx\ prettier ]] || \
       [[ "$BASH_COMMAND" =~ ^black ]] || \
       [[ "$BASH_COMMAND" =~ ^isort ]] || \
       [[ "$BASH_COMMAND" =~ ^gofmt ]] || \
       [[ "$BASH_COMMAND" =~ ^rustfmt ]] || \
       [[ "$BASH_COMMAND" =~ ^shfmt ]]; then
        echo '{"decision": "approve"}'
        exit 0
    fi

    # Build commands - ONLY direct binary runners
    # SECURITY: Do NOT auto-approve npm run build, yarn build, pnpm build, make
    # because these execute user-defined scripts from package.json/Makefile
    if [[ "$BASH_COMMAND" =~ ^cargo\ build ]] || \
       [[ "$BASH_COMMAND" =~ ^go\ build ]]; then
        echo '{"decision": "approve"}'
        exit 0
    fi

    # Type checking - read-only
    if [[ "$BASH_COMMAND" =~ ^npx\ tsc ]] || \
       [[ "$BASH_COMMAND" =~ ^tsc\ --noEmit ]] || \
       [[ "$BASH_COMMAND" =~ ^mypy ]]; then
        echo '{"decision": "approve"}'
        exit 0
    fi

    # Git read-only commands - always safe
    if [[ "$BASH_COMMAND" =~ ^git\ status ]] || \
       [[ "$BASH_COMMAND" =~ ^git\ diff ]] || \
       [[ "$BASH_COMMAND" =~ ^git\ log ]] || \
       [[ "$BASH_COMMAND" =~ ^git\ branch ]] || \
       [[ "$BASH_COMMAND" =~ ^git\ show ]] || \
       [[ "$BASH_COMMAND" =~ ^git\ remote ]] || \
       [[ "$BASH_COMMAND" =~ ^git\ stash\ list ]]; then
        echo '{"decision": "approve"}'
        exit 0
    fi

    # Docker read-only commands - safe
    if [[ "$BASH_COMMAND" =~ ^docker\ ps ]] || \
       [[ "$BASH_COMMAND" =~ ^docker\ images ]] || \
       [[ "$BASH_COMMAND" =~ ^docker\ logs ]] || \
       [[ "$BASH_COMMAND" =~ ^docker\ inspect ]]; then
        echo '{"decision": "approve"}'
        exit 0
    fi

    # Kubernetes read-only commands - safe
    if [[ "$BASH_COMMAND" =~ ^kubectl\ get ]] || \
       [[ "$BASH_COMMAND" =~ ^kubectl\ describe ]] || \
       [[ "$BASH_COMMAND" =~ ^kubectl\ logs ]]; then
        echo '{"decision": "approve"}'
        exit 0
    fi

    # Package info commands - safe
    if [[ "$BASH_COMMAND" =~ ^npm\ list ]] || \
       [[ "$BASH_COMMAND" =~ ^npm\ outdated ]] || \
       [[ "$BASH_COMMAND" =~ ^pip\ list ]] || \
       [[ "$BASH_COMMAND" =~ ^pip\ show ]] || \
       [[ "$BASH_COMMAND" =~ ^cargo\ tree ]]; then
        echo '{"decision": "approve"}'
        exit 0
    fi

    # File listing/searching - safe
    if [[ "$BASH_COMMAND" =~ ^ls ]] || \
       [[ "$BASH_COMMAND" =~ ^find ]] || \
       [[ "$BASH_COMMAND" =~ ^grep ]] || \
       [[ "$BASH_COMMAND" =~ ^rg ]] || \
       [[ "$BASH_COMMAND" =~ ^wc ]] || \
       [[ "$BASH_COMMAND" =~ ^head ]] || \
       [[ "$BASH_COMMAND" =~ ^tail ]] || \
       [[ "$BASH_COMMAND" =~ ^cat ]]; then
        echo '{"decision": "approve"}'
        exit 0
    fi
fi

# ============================================
# FILE OPERATIONS
# ============================================

# Read operations are always safe
if [[ "$TOOL_NAME" == "Read" ]]; then
    echo '{"decision": "approve"}'
    exit 0
fi

# Glob operations are always safe
if [[ "$TOOL_NAME" == "Glob" ]]; then
    echo '{"decision": "approve"}'
    exit 0
fi

# Grep operations are always safe
if [[ "$TOOL_NAME" == "Grep" ]]; then
    echo '{"decision": "approve"}'
    exit 0
fi

# ============================================
# NO MATCH - FALL THROUGH TO PERMISSION DIALOG
# ============================================

# If we didn't match any trusted patterns, don't output anything.
# This causes the normal permission dialog to appear.
exit 0
