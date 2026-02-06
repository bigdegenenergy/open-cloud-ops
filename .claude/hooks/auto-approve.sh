#!/bin/bash
# PermissionRequest Hook - Auto-Approve Trusted Commands
# Eliminates approval friction for commands you already trust.
#
# Output: JSON with decision field
#   {"decision": "approve"} - Auto-approve the command
#   {"decision": "deny", "message": "reason"} - Block the command
#   (no output) - Fall through to normal permission dialog
#
# Exit codes:
#   0 = Hook ran successfully (output determines action)
#   non-zero = Hook failed, fall through to normal behavior

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
# This prevents attacks like "npm test; rm -rf /" or "npm test && malicious"
contains_shell_metacharacters() {
    local cmd="$1"

    # Define forbidden patterns in variables for clarity and safety
    # Pattern 1: Command chaining operators (;, &, |)
    local CHAIN_CHARS='[;&|]'
    # Pattern 2: Command substitution (backticks or $())
    local CMD_SUBST='(`|\$\()'
    # Pattern 3: Output redirection (>)
    local REDIRECT='>'
    # Pattern 4: Newlines (critical - "npm test\nrm -rf /" bypass)
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
# SECURITY PRINCIPLES:
# 1. Only auto-approve commands with fixed behavior (not project-defined)
# 2. Commands that execute project scripts (npm test, make) are NOT auto-approved
#    because the agent can edit those scripts and inject arbitrary code
# 3. File read commands are NOT auto-approved (they could read ~/.ssh/*, /etc/*)
#    The Read/Glob/Grep tools are preferred and auto-approved below

if [[ "$TOOL_NAME" == "Bash" ]] && [[ -n "$BASH_COMMAND" ]]; then
    # SECURITY: Never auto-approve commands with shell metacharacters
    if contains_shell_metacharacters "$BASH_COMMAND"; then
        # Fall through to permission dialog for safety
        exit 0
    fi

    # Compiler/runtime test commands with FIXED behavior (not project-defined scripts)
    # These run the language's built-in test runner, not user scripts:
    if [[ "$BASH_COMMAND" =~ ^pytest ]] || \
       [[ "$BASH_COMMAND" =~ ^python\ -m\ pytest ]] || \
       [[ "$BASH_COMMAND" =~ ^cargo\ test ]] || \
       [[ "$BASH_COMMAND" =~ ^go\ test ]]; then
        echo '{"decision": "approve"}'
        exit 0
    fi

    # Lint commands - read-only analysis with fixed behavior
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

    # Format commands - fixed behavior formatters
    if [[ "$BASH_COMMAND" =~ ^npx\ prettier ]] || \
       [[ "$BASH_COMMAND" =~ ^black ]] || \
       [[ "$BASH_COMMAND" =~ ^isort ]] || \
       [[ "$BASH_COMMAND" =~ ^gofmt ]] || \
       [[ "$BASH_COMMAND" =~ ^rustfmt ]] || \
       [[ "$BASH_COMMAND" =~ ^shfmt ]]; then
        echo '{"decision": "approve"}'
        exit 0
    fi

    # Compiler/build commands with FIXED behavior (not project-defined scripts)
    if [[ "$BASH_COMMAND" =~ ^cargo\ build ]] || \
       [[ "$BASH_COMMAND" =~ ^go\ build ]]; then
        echo '{"decision": "approve"}'
        exit 0
    fi

    # Type checking - read-only with fixed behavior
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

    # Package info commands - safe (read-only metadata)
    if [[ "$BASH_COMMAND" =~ ^npm\ list ]] || \
       [[ "$BASH_COMMAND" =~ ^npm\ outdated ]] || \
       [[ "$BASH_COMMAND" =~ ^pip\ list ]] || \
       [[ "$BASH_COMMAND" =~ ^pip\ show ]] || \
       [[ "$BASH_COMMAND" =~ ^cargo\ tree ]]; then
        echo '{"decision": "approve"}'
        exit 0
    fi

    # NOTE: The following are intentionally NOT auto-approved because they
    # execute project-defined scripts that the agent could modify:
    #   npm test, npm run build, npm run lint, pnpm *, yarn *
    #   make, make test, make build
    # File read commands (ls, cat, find, grep, head, tail) are also NOT
    # auto-approved because they can access files outside the repository.
    # Use the Read/Glob/Grep tools instead (auto-approved below).
fi

# ============================================
# FILE OPERATIONS (Claude Code native tools)
# ============================================
# These tools are sandboxed by Claude Code and only access
# files within the project directory.

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
