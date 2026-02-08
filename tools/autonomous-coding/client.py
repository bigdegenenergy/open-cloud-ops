"""
Claude SDK Client Configuration
===============================

Functions for creating and configuring the Claude Agent SDK client.
"""

import json
import os
from pathlib import Path

from claude_code_sdk import ClaudeCodeOptions, ClaudeSDKClient
from claude_code_sdk.types import HookMatcher

from security import bash_security_hook


# Puppeteer MCP tools for browser automation
PUPPETEER_TOOLS = [
    "mcp__puppeteer__puppeteer_navigate",
    "mcp__puppeteer__puppeteer_screenshot",
    "mcp__puppeteer__puppeteer_click",
    "mcp__puppeteer__puppeteer_fill",
    "mcp__puppeteer__puppeteer_select",
    "mcp__puppeteer__puppeteer_hover",
    "mcp__puppeteer__puppeteer_evaluate",
]

# Built-in tools
BUILTIN_TOOLS = [
    "Read",
    "Write",
    "Edit",
    "Glob",
    "Grep",
    "Bash",
]


def create_client(project_dir: Path, model: str) -> ClaudeSDKClient:
    """
    Create a Claude Agent SDK client with multi-layered security.

    Args:
        project_dir: Directory for the project
        model: Claude model to use

    Returns:
        Configured ClaudeSDKClient

    Security layers (defense in depth):
    1. Sandbox - OS-level bash command isolation prevents filesystem escape
    2. Permissions - File operations restricted to project_dir only
    3. Security hooks - Bash commands validated against an allowlist
       (see security.py for ALLOWED_COMMANDS)
    """
    api_key = os.environ.get("ANTHROPIC_API_KEY")
    if not api_key:
        raise ValueError(
            "ANTHROPIC_API_KEY environment variable not set.\n"
            "Get your API key from: https://console.anthropic.com/"
        )

    # CRITICAL SECURITY: Compute absolute path to security module
    # This prevents the agent from modifying its own security hooks
    # regardless of where project_dir is located
    security_module_dir = str(Path(__file__).resolve().parent)

    # Create comprehensive security settings
    # Note: Using relative paths ("./**") restricts access to project directory
    # since cwd is set to project_dir
    security_settings = {
        "sandbox": {"enabled": True, "autoAllowBashIfSandboxed": True},
        "permissions": {
            "defaultMode": "acceptEdits",  # Auto-approve edits within allowed directories
            "allow": [
                # Allow all file operations within the project directory
                "Read(./**)",
                "Write(./**)",
                "Edit(./**)",
                "Glob(./**)",
                "Grep(./**)",
                # Bash permission granted here, but actual commands are validated
                # by the bash_security_hook (see security.py for allowed commands)
                "Bash(*)",
                # Allow Puppeteer MCP tools for browser automation
                *PUPPETEER_TOOLS,
            ],
            # CRITICAL: Deny write/edit to security modules to prevent bypass attacks
            # Uses absolute path to work regardless of project_dir location
            # Agent cannot modify the security.py module or hooks
            "deny": [
                f"Write({security_module_dir}/**)",
                f"Edit({security_module_dir}/**)",
            ],
        },
    }

    # Ensure project directory exists before creating settings file
    project_dir.mkdir(parents=True, exist_ok=True)

    # Create a backup/checkpoint before auto-approving edits
    # This allows recovery if the agent makes destructive changes
    try:
        import subprocess
        from datetime import datetime
        # Create a git checkpoint if we're in a git repo
        result = subprocess.run(
            ["git", "rev-parse", "--git-dir"],
            cwd=project_dir,
            capture_output=True,
            timeout=5
        )
        if result.returncode == 0:
            # Git repo exists - create timestamped checkpoint to preserve first 'clean' state
            try:
                # Check if there are uncommitted changes
                status_result = subprocess.run(
                    ["git", "status", "--porcelain"],
                    cwd=project_dir,
                    capture_output=True,
                    timeout=5,
                    text=True
                )
                
                stash_applied = False
                if status_result.stdout.strip():
                    # Uncommitted changes exist - stash them before checkpoint
                    stash_result = subprocess.run(
                        ["git", "stash", "push", "-u", "-m", f"pre-agent-{datetime.now().isoformat()}"],
                        cwd=project_dir,
                        capture_output=True,
                        timeout=5
                    )
                    if stash_result.returncode == 0:
                        stash_applied = True
                        print("   - Stashed uncommitted changes (recoverable via 'git stash pop')")
                
                # Use timestamp to ensure each run gets its own checkpoint
                timestamp = datetime.now().strftime("%Y%m%d-%H%M%S")
                checkpoint_branch = f"checkpoint/pre-agent-{timestamp}"
                subprocess.run(
                    ["git", "branch", checkpoint_branch],
                    cwd=project_dir,
                    capture_output=True,
                    timeout=5,
                    check=True
                )
                # Also create/update a "latest" checkpoint for convenience
                subprocess.run(
                    ["git", "branch", "-f", "checkpoint/pre-agent-latest"],
                    cwd=project_dir,
                    capture_output=True,
                    timeout=5,
                    check=False  # Don't fail if branch already exists
                )
                # Don't switch branches - just mark the checkpoint for recovery
                # User can manually recover using either:
                # 1. Latest: git reset --hard checkpoint/pre-agent-latest
                # 2. Specific run: git reset --hard checkpoint/pre-agent-<timestamp>
                # Print the actual branch name for reference
                print(f"   - Git checkpoint: {checkpoint_branch} (also available as checkpoint/pre-agent-latest)")
            except Exception:
                pass  # Silently fail if git operations don't work
    except Exception:
        pass  # Silently continue if git is not available

    # Write settings to a file in the project directory
    settings_file = project_dir / ".claude_settings.json"
    with open(settings_file, "w") as f:
        json.dump(security_settings, f, indent=2)

    print(f"Created security settings at {settings_file}")
    print("   - Sandbox enabled (OS-level bash isolation)")
    print(f"   - Filesystem restricted to: {project_dir.resolve()}")
    print("   - Bash commands restricted to allowlist (see security.py)")
    print("   - MCP servers: puppeteer (browser automation)")
    print("   - Auto-approval: Enabled with git checkpoint backup")
    print("   - Recovery: If needed, run: git reset --hard checkpoint/pre-agent-latest")
    print()

    return ClaudeSDKClient(
        options=ClaudeCodeOptions(
            model=model,
            system_prompt="You are an expert full-stack developer building a production-quality web application.",
            allowed_tools=[
                *BUILTIN_TOOLS,
                *PUPPETEER_TOOLS,
            ],
            mcp_servers={
                # Launch MCP server from its installation directory (tools/autonomous-coding/)
                # Uses --prefix to point npm to where puppeteer-mcp-server is installed
                # This ensures the server runs from the correct directory regardless of project_dir
                "puppeteer": {
                    "command": "npm",
                    "args": ["--prefix", security_module_dir, "exec", "puppeteer-mcp-server"]
                }
            },
            hooks={
                "PreToolUse": [
                    HookMatcher(matcher="Bash", hooks=[bash_security_hook]),
                ],
            },
            max_turns=1000,
            cwd=str(project_dir.resolve()),
            settings=str(settings_file.resolve()),  # Use absolute path
        )
    )
