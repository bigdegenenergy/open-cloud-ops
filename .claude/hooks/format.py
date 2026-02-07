#!/usr/bin/env python3
"""
The Janitor: Auto-format code after every edit.

This hook runs after Write/Edit operations and applies the appropriate
formatter based on file type. It reads Claude's tool input from stdin
to determine which file was modified.

Supported formatters:
- Prettier: JS, TS, JSX, TSX, JSON, MD, CSS, HTML, Vue, Svelte
- Black + isort: Python
- gofmt: Go
- rustfmt: Rust
- rubocop: Ruby
- shfmt: Shell scripts
"""

import json
import os
import subprocess
import sys
from pathlib import Path


def format_file(file_path: str) -> None:
    """Apply the appropriate formatter based on file extension."""
    if not os.path.exists(file_path):
        return

    path = Path(file_path)
    ext = path.suffix.lower()

    try:
        # JavaScript/TypeScript/Web -> Prettier
        if ext in (
            ".js",
            ".ts",
            ".tsx",
            ".jsx",
            ".json",
            ".md",
            ".css",
            ".html",
            ".vue",
            ".svelte",
        ):
            subprocess.run(
                ["npx", "prettier", "--write", file_path],
                stderr=subprocess.DEVNULL,
                stdout=subprocess.DEVNULL,
                timeout=30,
            )

        # Python -> Black + isort
        elif ext == ".py":
            subprocess.run(
                ["black", "--quiet", file_path],
                stderr=subprocess.DEVNULL,
                stdout=subprocess.DEVNULL,
                timeout=30,
            )
            subprocess.run(
                ["isort", "--quiet", file_path],
                stderr=subprocess.DEVNULL,
                stdout=subprocess.DEVNULL,
                timeout=30,
            )

        # Go -> gofmt
        elif ext == ".go":
            subprocess.run(
                ["gofmt", "-w", file_path],
                stderr=subprocess.DEVNULL,
                stdout=subprocess.DEVNULL,
                timeout=30,
            )

        # Rust -> rustfmt
        elif ext == ".rs":
            subprocess.run(
                ["rustfmt", file_path],
                stderr=subprocess.DEVNULL,
                stdout=subprocess.DEVNULL,
                timeout=30,
            )

        # Ruby -> rubocop
        elif ext == ".rb":
            subprocess.run(
                ["rubocop", "-a", file_path],
                stderr=subprocess.DEVNULL,
                stdout=subprocess.DEVNULL,
                timeout=30,
            )

        # Shell -> shfmt
        elif ext in (".sh", ".bash"):
            subprocess.run(
                ["shfmt", "-w", file_path],
                stderr=subprocess.DEVNULL,
                stdout=subprocess.DEVNULL,
                timeout=30,
            )

    except (subprocess.TimeoutExpired, FileNotFoundError):
        # Formatter not installed or timed out - fail silently
        pass


def main():
    try:
        # Read Claude's tool input from stdin
        input_data = json.load(sys.stdin)

        # Extract file path from tool_input
        file_path = input_data.get("tool_input", {}).get("file_path")

        if file_path:
            format_file(file_path)

    except (json.JSONDecodeError, KeyError):
        # Invalid input - fail silently
        pass
    except Exception:
        # Catch-all to never block the agent
        pass


if __name__ == "__main__":
    main()
