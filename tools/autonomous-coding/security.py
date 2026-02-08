"""
Security Hooks for Autonomous Coding Agent
==========================================

Pre-tool-use hooks that validate bash commands for security.
Uses an allowlist approach - only explicitly permitted commands can run.

SECURITY BOUNDARY:
  This module provides defense-in-depth against shell-injection attacks
  (e.g., preventing 'npm run test -- $(malicious_code)').
  
  It is NOT a comprehensive security sandbox. The agent can still run
  arbitrary code via allowed commands like python3 (e.g., 'python3 -c ...').
  
  For production deployments, run the agent in a containerized environment
  or isolated VM with restricted network/filesystem access.
"""

import os
import shlex


# Allowed commands for development tasks
# Minimal set needed for the autonomous coding demo
ALLOWED_COMMANDS = {
    # File inspection
    "ls",
    "cat",
    "head",
    "tail",
    "wc",
    "grep",
    # File operations (agent uses SDK tools for most file ops, but cp/mkdir needed occasionally)
    "cp",
    "mkdir",
    "chmod",  # For making scripts executable; validated separately
    # Directory
    "pwd",
    # Node.js development
    "npm",
    "node",
    # Version control
    "git",
    # Process management
    "ps",
    "lsof",
    "sleep",
    "pkill",  # For killing dev servers; validated separately
    # Script execution
    "init.sh",  # Init scripts; validated separately
}

# Commands that need additional validation even when in the allowlist
COMMANDS_NEEDING_EXTRA_VALIDATION = {"pkill", "chmod", "init.sh"}


def split_command_segments(command_string: str) -> list[str]:
    """
    Split a compound command into individual command segments.

    Handles command chaining (&&, ||, ;) and compound operators (&, (, ), {, }).
    Uses shlex with punctuation_chars to properly tokenize shell operators.

    Args:
        command_string: The full shell command

    Returns:
        List of individual command segments
    """
    try:
        # Use shlex with punctuation_chars to ensure operators are separate tokens
        # This prevents "npm;rm" from being parsed as a single token
        lexer = shlex.shlex(command_string, posix=True, punctuation_chars='|&;(){}<>')
        tokens = list(lexer)
    except ValueError:
        # Malformed command - return as-is for validation to catch
        return [command_string]

    segments = []
    current_segment = []

    for token in tokens:
        # Skip whitespace-only tokens
        if not token or token.isspace():
            continue
            
        # Shell operators and special characters indicate segment boundaries
        if token in ("|", "||", "&&", ";", "&", "(", ")", "{", "}"):
            if current_segment:
                # Reconstruct the segment from tokens
                segments.append(" ".join(current_segment))
                current_segment = []
        else:
            current_segment.append(token)

    # Add final segment if any
    if current_segment:
        segments.append(" ".join(current_segment))

    return segments


def extract_commands(command_string: str) -> list[str]:
    """
    Extract command names from a shell command string.

    Handles pipes, command chaining (&&, ||, ;), and subshells.
    Uses shlex with punctuation_chars to properly tokenize operators.
    Returns the base command names (without paths).

    Args:
        command_string: The full shell command

    Returns:
        List of command names found in the string
    """
    try:
        # Use shlex with punctuation_chars to ensure operators are separate tokens
        lexer = shlex.shlex(command_string, posix=True, punctuation_chars='|&;(){}<>')
        tokens = list(lexer)
    except ValueError:
        # Malformed command (unclosed quotes, etc.)
        # Return empty to trigger block (fail-safe)
        return []

    commands = []
    expect_command = True

    for token in tokens:
        # Skip whitespace
        if not token or token.isspace():
            continue
            
        # Shell operators and special characters indicate a new command follows
        if token in ("|", "||", "&&", "&", ";", "(", ")", "{", "}"):
            expect_command = True
            continue

        # Skip shell keywords that precede commands
        if token in (
            "if",
            "then",
            "else",
            "elif",
            "fi",
            "for",
            "while",
            "until",
            "do",
            "done",
            "case",
            "esac",
            "in",
            "!",
            "{",
            "}",
        ):
            continue

        # Skip flags/options
        if token.startswith("-"):
            continue

        # Skip variable assignments (VAR=value)
        if "=" in token and not token.startswith("="):
            continue

        if expect_command:
            # Extract the base command name (handle paths like /usr/bin/python)
            cmd = os.path.basename(token)
            commands.append(cmd)
            expect_command = False

    return commands


def validate_pkill_command(command_string: str) -> tuple[bool, str]:
    """
    Validate pkill commands - only allow killing dev-related processes.

    Uses shlex to parse the command, avoiding regex bypass vulnerabilities.

    Returns:
        Tuple of (is_allowed, reason_if_blocked)
    """
    # Allowed process names for pkill
    allowed_process_names = {
        "node",
        "npm",
        "npx",
        "vite",
        "next",
    }

    try:
        tokens = shlex.split(command_string, posix=True)
    except ValueError:
        return False, "Could not parse pkill command"

    if not tokens:
        return False, "Empty pkill command"

    # Separate flags from arguments
    args = []
    for token in tokens[1:]:
        if not token.startswith("-"):
            args.append(token)

    if not args:
        return False, "pkill requires a process name"

    # The target is typically the last non-flag argument
    target = args[-1]

    # For -f flag (full command line match), extract the first word as process name
    # e.g., "pkill -f 'node server.js'" -> target is "node server.js", process is "node"
    if " " in target:
        target = target.split()[0]

    if target in allowed_process_names:
        return True, ""
    return False, f"pkill only allowed for dev processes: {allowed_process_names}"


def validate_chmod_command(command_string: str) -> tuple[bool, str]:
    """
    Validate chmod commands - only allow making files executable with +x.
    
    Restrictions:
    - Only +x mode allowed (making files executable)
    - Recursive operations (-R, --recursive) not allowed (prevents bulk permission changes)
    - No other flags or mode changes permitted

    Returns:
        Tuple of (is_allowed, reason_if_blocked)
    """
    try:
        tokens = shlex.split(command_string, posix=True)
    except ValueError:
        return False, "Could not parse chmod command"

    if not tokens or tokens[0] != "chmod":
        return False, "Not a chmod command"

    # Strict validation: chmod can only have mode and file(s)
    # Pattern: chmod <mode> <file1> [<file2> ...]
    # No flags allowed, exactly one mode token
    
    if len(tokens) < 3:
        return False, "chmod requires: chmod <mode> <file> [files...]"

    # Token 0 is "chmod"
    # Token 1 is the mode
    # Tokens 2+ are files
    
    mode = tokens[1]
    files = tokens[2:]

    # Check for any flag tokens (starting with -)
    # This would allow -R, --recursive, etc.
    if mode.startswith("-"):
        return False, f"chmod flags not allowed: {mode}"

    # Check files for invalid content
    for f in files:
        if f.startswith("-"):
            return False, f"Invalid argument (flag): {f}"

    # Only allow +x variants (making files executable)
    # This matches: +x, u+x, g+x, o+x, a+x, ug+x, etc.
    # NOT: 777, 755, 644, or other octal modes
    # NOT: multiple modes like "777 +x" which mixes octal with symbolic
    import re

    if not re.match(r"^[ugoa]*\+x$", mode):
        return False, f"chmod only allowed with +x mode, got: {mode}"

    return True, ""


def validate_init_script(command_string: str) -> tuple[bool, str]:
    """
    Validate init.sh script execution - only allow ./init.sh.

    Prevents execution of init.sh from other directories (e.g., /tmp/init.sh)
    which would bypass project-directory sandbox.

    Returns:
        Tuple of (is_allowed, reason_if_blocked)
    """
    try:
        tokens = shlex.split(command_string, posix=True)
    except ValueError:
        return False, "Could not parse init script command"

    if not tokens:
        return False, "Empty command"

    # The command must be exactly ./init.sh (with optional arguments)
    script = tokens[0]

    # Only allow ./init.sh (project-relative, current directory)
    if script == "./init.sh":
        return True, ""

    return False, f"Only ./init.sh is allowed (project-relative), got: {script}"


def validate_arguments_for_injection(tokens: list[str]) -> tuple[bool, str]:
    """
    Validate that arguments don't contain unquoted shell operators that enable injection.

    Key insight: shlex.split(posix=True, punctuation_chars='|&;(){}<>') separates
    operators as individual tokens ONLY when they're unquoted. Operators inside
    quoted strings (e.g., 'http://api.com?key=1&val=2') stay as a single token.

    Rules:
    1. Reject tokens that ARE EXACTLY operators (e.g., token == '&' means unquoted)
    2. Reject command substitution patterns at token boundaries (indicates unquoting)

    Args:
        tokens: List of command tokens (parsed by shlex with punctuation_chars)

    Returns:
        Tuple of (is_allowed, reason_if_blocked)
    """
    # Operators that should NEVER appear as standalone tokens
    # (shlex with punctuation_chars separates these ONLY when unquoted)
    # If a token IS an operator, it was not inside quotes
    dangerous_operators = ("|", "||", "&&", "&", ";", "(", ")", "{", "}", "<", ">", ">>")

    # Command substitution and variable expansion patterns that indicate code execution
    # Only dangerous at token start (indicates it was unquoted)
    # Includes: $(), `...`, ${...} (variable expansion can be exploited in some contexts)
    command_sub_markers = ["$(", "`", "${"]

    # Check all tokens
    for token in tokens:
        # Rule 1: Token is EXACTLY an operator
        # shlex guarantees operators are isolated tokens only if unquoted
        # Example: ['curl', 'http://api.com?a=1&b=2', 'file'] — the & is inside token
        #          vs ['curl', '&', 'whoami'] — the & is a separate token
        if token in dangerous_operators:
            return False, f"Unquoted shell operator '{token}' found in command"

        # Rule 2: Token starts/ends with command substitution
        # If $( or backtick appears at token boundary, it indicates unquoting
        # Safe: 'url?key=$(value)' (was quoted, shlex keeps as one token)
        # Danger: '$(cmd)' (unquoted, separate token or at token start)
        for marker in command_sub_markers:
            if token.startswith(marker):
                return False, f"Command substitution '{marker}' at token start (unquoted code execution): {token}"

    return True, ""


def get_command_for_validation(cmd: str, segments: list[str]) -> str:
    """
    Find the specific command segment that contains the given command.

    Args:
        cmd: The command name to find
        segments: List of command segments

    Returns:
        The segment containing the command, or empty string if not found
    """
    for segment in segments:
        segment_commands = extract_commands(segment)
        if cmd in segment_commands:
            return segment
    return ""


async def bash_security_hook(input_data, tool_use_id=None, context=None):
    """
    Pre-tool-use hook that validates bash commands using an allowlist.

    Only commands in ALLOWED_COMMANDS are permitted.
    Validates all arguments for shell injection patterns.

    Args:
        input_data: Dict containing tool_name and tool_input
        tool_use_id: Optional tool use ID
        context: Optional context

    Returns:
        Empty dict to allow, or {"decision": "block", "reason": "..."} to block
    """
    if input_data.get("tool_name") != "Bash":
        return {}

    command = input_data.get("tool_input", {}).get("command", "")
    if not command:
        return {}

    # Extract all commands from the command string
    commands = extract_commands(command)

    if not commands:
        # Could not parse - fail safe by blocking
        return {
            "decision": "block",
            "reason": f"Could not parse command for security validation: {command}",
        }

    # Split into segments for per-command validation
    segments = split_command_segments(command)

    # Check each command against the allowlist
    for cmd in commands:
        if cmd not in ALLOWED_COMMANDS:
            return {
                "decision": "block",
                "reason": f"Command '{cmd}' is not in the allowed commands list",
            }

        # Additional validation for sensitive commands
        if cmd in COMMANDS_NEEDING_EXTRA_VALIDATION:
            # Find the specific segment containing this command
            cmd_segment = get_command_for_validation(cmd, segments)
            if not cmd_segment:
                cmd_segment = command  # Fallback to full command

            if cmd == "pkill":
                allowed, reason = validate_pkill_command(cmd_segment)
                if not allowed:
                    return {"decision": "block", "reason": reason}
            elif cmd == "chmod":
                allowed, reason = validate_chmod_command(cmd_segment)
                if not allowed:
                    return {"decision": "block", "reason": reason}
            elif cmd == "init.sh":
                allowed, reason = validate_init_script(cmd_segment)
                if not allowed:
                    return {"decision": "block", "reason": reason}

        # Validate arguments in this command segment for injection patterns
        cmd_segment = get_command_for_validation(cmd, segments)
        if cmd_segment:
            try:
                tokens = shlex.split(cmd_segment, posix=True)
                allowed, reason = validate_arguments_for_injection(tokens)
                if not allowed:
                    return {"decision": "block", "reason": reason}
            except ValueError:
                # If we can't parse, fail safe
                return {
                    "decision": "block",
                    "reason": f"Could not parse arguments for injection validation in: {cmd_segment}",
                }

    return {}
