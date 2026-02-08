#!/usr/bin/env python3
"""
Gateway Command Mapping

Maps chat platform commands to Claude Code commands.
Used by webhook handlers to translate incoming requests.

SECURITY WARNING:
================
The permission system relies on user_id provided in the webhook payload.
Since repository_dispatch often uses a shared token, any holder of the token
can potentially spoof the user_id to bypass admin restrictions.

REQUIRED SECURITY MEASURES:
1. Validate the payload signature from the chat platform if possible
2. Implement strict access controls on the webhook Personal Access Token (PAT)
3. Ensure users.json is NOT writable by the GitHub Actions workflow
4. Consider implementing additional authentication layers for admin commands
5. Review webhook audit logs regularly for unauthorized access attempts

The current implementation assumes trusted webhook sources. For production use,
implement proper cryptographic signature validation for incoming webhooks.
"""

from dataclasses import dataclass
from enum import Enum
import hashlib
import hmac
import os
from typing import Optional


class PermissionLevel(Enum):
    """User permission levels for command execution."""

    VIEWER = "viewer"
    MEMBER = "member"
    ADMIN = "admin"


class WebhookVerificationError(Exception):
    """Raised when webhook signature verification fails."""

    pass


def verify_webhook_signature(
    payload: bytes,
    signature: str,
    secret: Optional[str] = None,
    raise_on_missing_secret: bool = False,
) -> bool:
    """
    Verify the HMAC signature of a webhook payload.

    SECURITY: This function implements FAIL-CLOSED behavior by default.
    If WEBHOOK_SECRET is not configured, verification returns False.
    Set raise_on_missing_secret=True for strict mode that raises an exception.

    Args:
        payload: The raw webhook payload bytes
        signature: The signature from the webhook header (e.g., "sha256=...")
        secret: Optional secret override; if None, uses WEBHOOK_SECRET env var
        raise_on_missing_secret: If True, raise WebhookVerificationError when
            secret is not configured. If False (default), return False for
            backward compatibility.

    Returns:
        True if signature is valid, False otherwise

    Raises:
        WebhookVerificationError: If secret is not configured AND
            raise_on_missing_secret=True
    """
    # Get secret from environment if not provided
    webhook_secret = secret or os.environ.get("WEBHOOK_SECRET")

    # SECURITY: Fail-closed - if no secret configured, reject the request
    # This prevents accidental exposure when secret is missing
    if not webhook_secret:
        if raise_on_missing_secret:
            raise WebhookVerificationError(
                "WEBHOOK_SECRET environment variable is not set. "
                "Webhook verification cannot proceed without a secret. "
                "This is a security control - webhooks must be authenticated."
            )
        # Return False for backward compatibility (still fail-closed, but no exception)
        return False

    # Parse signature format (e.g., "sha256=abc123...")
    if "=" not in signature:
        return False

    algorithm, provided_signature = signature.split("=", 1)

    # Only support SHA-256 for security
    if algorithm.lower() != "sha256":
        return False

    # Compute expected signature
    expected_signature = hmac.new(
        webhook_secret.encode("utf-8"),
        payload,
        hashlib.sha256,
    ).hexdigest()

    # Constant-time comparison to prevent timing attacks
    return hmac.compare_digest(expected_signature, provided_signature)


@dataclass
class CommandMapping:
    """Maps a chat command to a Claude command."""

    chat_pattern: str  # Pattern to match (e.g., "plan", "qa")
    claude_command: str  # Claude command to execute (e.g., "/plan", "/qa")
    description: str  # Human-readable description
    required_permission: PermissionLevel
    accepts_args: bool = True  # Whether the command accepts arguments


# Command mapping registry
COMMAND_MAPPINGS: dict[str, CommandMapping] = {
    "plan": CommandMapping(
        chat_pattern="plan",
        claude_command="/plan",
        description="Plan a feature implementation",
        required_permission=PermissionLevel.MEMBER,
        accepts_args=True,
    ),
    "qa": CommandMapping(
        chat_pattern="qa",
        claude_command="/qa",
        description="Run tests and fix failures",
        required_permission=PermissionLevel.MEMBER,
        accepts_args=False,
    ),
    "review": CommandMapping(
        chat_pattern="review",
        claude_command="/review",
        description="Critical code review",
        required_permission=PermissionLevel.MEMBER,
        accepts_args=True,
    ),
    "zeno": CommandMapping(
        chat_pattern="zeno",
        claude_command="/zeno",
        description="Surgical code analysis with citations",
        required_permission=PermissionLevel.MEMBER,
        accepts_args=True,
    ),
    "ship": CommandMapping(
        chat_pattern="ship",
        claude_command="/ship",
        description="Commit and create PR",
        required_permission=PermissionLevel.ADMIN,
        accepts_args=True,
    ),
    "simplify": CommandMapping(
        chat_pattern="simplify",
        claude_command="/simplify",
        description="Clean up and refactor code",
        required_permission=PermissionLevel.MEMBER,
        accepts_args=True,
    ),
    "deslop": CommandMapping(
        chat_pattern="deslop",
        claude_command="/deslop",
        description="Aggressive code simplification",
        required_permission=PermissionLevel.MEMBER,
        accepts_args=True,
    ),
    "debug": CommandMapping(
        chat_pattern="debug",
        claude_command="/systematic-debug",
        description="Systematic bug investigation",
        required_permission=PermissionLevel.MEMBER,
        accepts_args=True,
    ),
    "status": CommandMapping(
        chat_pattern="status",
        claude_command="/gateway-status",
        description="Check current session status",
        required_permission=PermissionLevel.VIEWER,
        accepts_args=False,
    ),
    "approve": CommandMapping(
        chat_pattern="approve",
        claude_command="/workflow-approve",
        description="Approve pending workflow gate",
        required_permission=PermissionLevel.ADMIN,
        accepts_args=True,
    ),
    "help": CommandMapping(
        chat_pattern="help",
        claude_command="",
        description="Show available commands",
        required_permission=PermissionLevel.VIEWER,
        accepts_args=False,
    ),
}


def parse_command(message: str) -> tuple[Optional[str], Optional[str]]:
    """
    Parse a chat message to extract command and arguments.

    Args:
        message: Raw message from chat (e.g., "/claude plan add user auth")

    Returns:
        Tuple of (command_name, arguments) or (None, None) if not a command
    """
    # Maximum length limits to prevent DoS and injection attacks
    MAX_MESSAGE_LENGTH = 2000
    MAX_COMMAND_LENGTH = 100
    MAX_ARGS_LENGTH = 1500

    # Enforce maximum message length
    if len(message) > MAX_MESSAGE_LENGTH:
        message = message[:MAX_MESSAGE_LENGTH]

    # Remove common prefixes
    prefixes = ["/claude ", "!claude ", "@claude "]

    normalized = message.strip().lower()
    for prefix in prefixes:
        if normalized.startswith(prefix):
            message = message[len(prefix) :].strip()
            break
    else:
        # Not a claude command
        return None, None

    # Split into command and args
    parts = message.split(maxsplit=1)
    command = parts[0].lower() if parts else None
    args = parts[1] if len(parts) > 1 else None

    # Enforce length limits on parsed components
    if command and len(command) > MAX_COMMAND_LENGTH:
        command = command[:MAX_COMMAND_LENGTH]

    if args and len(args) > MAX_ARGS_LENGTH:
        args = args[:MAX_ARGS_LENGTH]

    return command, args


def get_command(name: str) -> Optional[CommandMapping]:
    """Get command mapping by name."""
    return COMMAND_MAPPINGS.get(name.lower())


def can_execute(command: CommandMapping, user_permission: PermissionLevel) -> bool:
    """
    Check if user has permission to execute command.

    SECURITY WARNING: This function assumes the user_permission parameter
    is trustworthy. In webhook scenarios, the user_id (and thus permission level)
    comes from the webhook payload and can be spoofed if the webhook is not
    properly authenticated. Always validate webhook signatures before trusting
    user_id claims.
    """
    permission_order = [
        PermissionLevel.VIEWER,
        PermissionLevel.MEMBER,
        PermissionLevel.ADMIN,
    ]
    user_level = permission_order.index(user_permission)
    required_level = permission_order.index(command.required_permission)
    return user_level >= required_level


def format_help() -> str:
    """Generate help text listing all available commands."""
    lines = ["**Available Commands**\n"]

    for name, cmd in COMMAND_MAPPINGS.items():
        perm = cmd.required_permission.value
        args = " <args>" if cmd.accepts_args else ""
        lines.append(f"- `/claude {name}{args}` - {cmd.description} ({perm})")

    return "\n".join(lines)


def _escape_xml(text: str) -> str:
    """
    Escape XML special characters to prevent injection attacks.

    Args:
        text: Text to escape

    Returns:
        Escaped text safe for use in XML/prompts
    """
    if not text:
        return text

    # Escape XML special characters
    replacements = {
        "<": "&lt;",
        ">": "&gt;",
        "&": "&amp;",
        '"': "&quot;",
        "'": "&apos;",
    }

    for char, escaped in replacements.items():
        text = text.replace(char, escaped)

    return text


def build_prompt(command: CommandMapping, args: Optional[str], context: dict) -> str:
    """
    Build the Claude prompt for the command.

    Args:
        command: The CommandMapping to execute
        args: Optional arguments from the chat message
        context: Additional context (repo, branch, user, etc.)

    Returns:
        Formatted prompt string for Claude
    """
    prompt_parts = [f"Execute: {command.claude_command}"]

    if args:
        # Escape args to prevent XML/prompt injection attacks
        safe_args = _escape_xml(args)
        prompt_parts.append(f"Arguments: {safe_args}")

    if context.get("repo"):
        safe_repo = _escape_xml(str(context["repo"]))
        prompt_parts.append(f"Repository: {safe_repo}")

    if context.get("branch"):
        safe_branch = _escape_xml(str(context["branch"]))
        prompt_parts.append(f"Branch: {safe_branch}")

    if context.get("pr_number"):
        # pr_number should be numeric, but sanitize anyway
        safe_pr = _escape_xml(str(context["pr_number"]))
        prompt_parts.append(f"PR: #{safe_pr}")

    return "\n".join(prompt_parts)


if __name__ == "__main__":
    # Test command parsing
    test_messages = [
        "/claude plan add user authentication",
        "!claude qa",
        "@claude review src/auth/",
        "/claude ship",
        "/claude help",
        "random message",
    ]

    print("Testing command parsing:\n")
    for msg in test_messages:
        cmd, args = parse_command(msg)
        print(f"  '{msg}'")
        print(f"    -> command: {cmd}, args: {args}")
        if cmd:
            mapping = get_command(cmd)
            if mapping:
                print(f"    -> Claude command: {mapping.claude_command}")
        print()
