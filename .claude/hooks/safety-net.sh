#!/bin/bash
# PreToolUse Hook - Safety Net
# Blocks dangerous commands before they execute
#
# Exit codes:
#   0 = Allow the action
#   2 = Block the action (message sent to stderr becomes error for agent)

# Read the tool input from stdin
INPUT=$(cat)

# Extract tool name and command (if Bash tool)
TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // empty')
BASH_COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

# ============================================
# DANGEROUS COMMAND PATTERNS
# ============================================

DANGEROUS_PATTERNS=(
    # Destructive file operations
    "rm -rf /"
    "rm -rf /*"
    "rm -rf ~"
    "rm -rf \$HOME"

    # Git destructive operations
    "git reset --hard"
    "git push.*--force"
    "git push.*-f"
    "git clean -fdx"

    # Database destructive operations
    "drop table"
    "drop database"
    "truncate table"
    "delete from.*where 1=1"
    "delete from.*without where"

    # System-level operations
    "chmod 777"
    "chmod -R 777"
    "sudo rm"
    "sudo chmod"
    "> /dev/sd"
    "mkfs"
    "dd if=.*/dev/"

    # Credential exposure
    "cat.*\.env"
    "cat.*credentials"
    "cat.*secret"
    "cat.*/etc/passwd"
    "cat.*/etc/shadow"
    "echo.*API_KEY"
    "echo.*SECRET"
    "echo.*PASSWORD"

    # Network exfiltration patterns
    "curl.*\|.*sh"
    "wget.*\|.*sh"
    "curl.*\|.*bash"
    "wget.*\|.*bash"
)

# ============================================
# SENSITIVE FILE PATTERNS
# ============================================

SENSITIVE_FILES=(
    ".env"
    ".env.local"
    ".env.production"
    "credentials.json"
    "secrets.yaml"
    "secrets.yml"
    ".ssh/id_rsa"
    ".ssh/id_ed25519"
    "*.pem"
    "*.key"
)

# ============================================
# CHECK BASH COMMANDS
# ============================================

if [[ "$TOOL_NAME" == "Bash" ]] && [[ -n "$BASH_COMMAND" ]]; then
    # Convert to lowercase for matching
    CMD_LOWER=$(echo "$BASH_COMMAND" | tr '[:upper:]' '[:lower:]')

    for pattern in "${DANGEROUS_PATTERNS[@]}"; do
        if echo "$CMD_LOWER" | grep -qiE "$pattern"; then
            echo "BLOCKED: Dangerous command detected." >&2
            echo "Pattern matched: $pattern" >&2
            echo "Command: $BASH_COMMAND" >&2
            echo "" >&2
            echo "This action violates safety protocols." >&2
            echo "If this is intentional, please run the command manually." >&2
            exit 2
        fi
    done
fi

# ============================================
# CHECK FILE ACCESS
# ============================================

if [[ "$TOOL_NAME" == "Read" || "$TOOL_NAME" == "Write" || "$TOOL_NAME" == "Edit" ]]; then
    if [[ -n "$FILE_PATH" ]]; then
        FILE_LOWER=$(echo "$FILE_PATH" | tr '[:upper:]' '[:lower:]')
        FILE_NAME=$(basename "$FILE_PATH")

        for pattern in "${SENSITIVE_FILES[@]}"; do
            # Check if filename matches pattern
            if [[ "$FILE_NAME" == $pattern ]] || [[ "$FILE_LOWER" == *"$pattern"* ]]; then
                echo "BLOCKED: Attempt to access sensitive file." >&2
                echo "File: $FILE_PATH" >&2
                echo "Pattern matched: $pattern" >&2
                echo "" >&2
                echo "Sensitive files should not be read or modified by agents." >&2
                echo "Handle credentials manually for security." >&2
                exit 2
            fi
        done
    fi
fi

# ============================================
# CHECK FOR SECRETS IN WRITE OPERATIONS
# ============================================

if [[ "$TOOL_NAME" == "Write" || "$TOOL_NAME" == "Edit" ]]; then
    CONTENT=$(echo "$INPUT" | jq -r '.tool_input.content // .tool_input.new_string // empty')

    if [[ -n "$CONTENT" ]]; then
        # Check for hardcoded secrets patterns
        if echo "$CONTENT" | grep -qiE "(password|api_key|secret|token)\s*[:=]\s*['\"][^'\"]{8,}['\"]"; then
            echo "WARNING: Potential hardcoded secret detected in content." >&2
            echo "File: $FILE_PATH" >&2
            echo "" >&2
            echo "Consider using environment variables instead." >&2
            # This is a warning, not a block - exit 0
        fi
    fi
fi

# ============================================
# ALL CHECKS PASSED
# ============================================

exit 0
