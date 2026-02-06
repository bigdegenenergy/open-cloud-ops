#!/usr/bin/env python3
"""
Commit Context Generator - Documents changes for PR review context.

Can be used in two modes:
1. Pre-commit hook (default): Analyzes staged changes
2. CI mode: Analyzes diff between two refs (e.g., base...head of a PR)

Usage:
    # Pre-commit hook (analyzes staged changes)
    python3 commit-context-generator.py

    # CI mode (analyzes PR diff)
    python3 commit-context-generator.py --base origin/main --head HEAD

    # Output to stdout only (for CI piping)
    python3 commit-context-generator.py --base $BASE --head $HEAD --stdout

Output:
- Prints context summary to stdout
- Saves full context to .claude/artifacts/commit-context.md (unless --stdout)
"""

from __future__ import annotations  # Enable PEP 604 syntax on Python 3.7+

import argparse
import json
import re
import subprocess
import sys
from datetime import datetime
from pathlib import Path


def run_git(args: list[str]) -> str:
    """Run a git command and return stdout."""
    result = subprocess.run(
        ["git"] + args,
        capture_output=True,
        text=True,
    )
    return result.stdout.strip()


def get_full_diff(base: str | None = None, head: str | None = None) -> str:
    """Get the full diff output.

    If base/head provided, compare those refs.
    Otherwise, use staged changes.
    Uses explicit noprefix=false to ensure consistent a/ b/ prefixes.
    """
    if base and head:
        return run_git(["-c", "diff.noprefix=false", "diff", f"{base}...{head}"])
    else:
        return run_git(["-c", "diff.noprefix=false", "diff", "--cached"])


def get_changed_files(base: str | None = None, head: str | None = None) -> list[str]:
    """Get list of changed files.

    If base/head provided, compare those refs.
    Otherwise, use staged changes.
    """
    if base and head:
        output = run_git(["diff", "--name-only", "--diff-filter=ACMRD", f"{base}...{head}"])
    else:
        output = run_git(["diff", "--cached", "--name-only", "--diff-filter=ACMRD"])
    return [f for f in output.split("\n") if f]


def parse_diff_by_file(full_diff: str) -> dict[str, str]:
    """Parse a unified diff into per-file diffs.

    Returns a dict mapping filepath to its diff content.
    Handles both regular files and renamed files.
    Also handles quoted filenames (for paths with spaces or special chars).
    """
    if not full_diff.strip():
        return {}

    file_diffs: dict[str, str] = {}
    current_file: str | None = None
    current_diff_lines: list[str] = []

    # Pattern to match diff headers
    # Handles: diff --git a/path/file b/path/file
    # Also handles renames: diff --git a/old/path b/new/path
    # Also handles quoted paths: diff --git "a/path with spaces" "b/path with spaces"
    diff_header_pattern = re.compile(r'^diff --git "?a/(.+?)"? "?b/(.+?)"?$')

    for line in full_diff.split("\n"):
        match = diff_header_pattern.match(line)
        if match:
            # Save previous file's diff
            if current_file and current_diff_lines:
                file_diffs[current_file] = "\n".join(current_diff_lines)

            # Start new file - use the 'b' path (destination) for renames
            current_file = match.group(2)
            current_diff_lines = [line]
        elif current_file is not None:
            current_diff_lines.append(line)

    # Save last file's diff
    if current_file and current_diff_lines:
        file_diffs[current_file] = "\n".join(current_diff_lines)

    return file_diffs


def categorize_file(filepath: str) -> str:
    """Categorize a file based on its path and extension."""
    path = Path(filepath)
    ext = path.suffix.lower()
    name = path.name.lower()
    parts = path.parts

    # Special files
    if name in ("readme.md", "readme.rst", "readme.txt", "readme"):
        return "documentation"
    if name in ("claude.md", "agents.md"):
        return "ai-config"
    if name in ("package.json", "pyproject.toml", "cargo.toml", "go.mod"):
        return "dependencies"
    if name in (".gitignore", ".env.example", "dockerfile", "docker-compose.yml"):
        return "configuration"

    # Test detection - check directory names and file prefixes/suffixes
    if (
        "tests" in parts
        or "test" in parts
        or "__tests__" in parts
        or name.startswith("test_")
        or name.endswith("_test.py")
        or name.endswith(".test.ts")
        or name.endswith(".test.js")
        or name.endswith(".spec.ts")
        or name.endswith(".spec.js")
    ):
        return "tests"

    # By directory
    if ".github" in parts:
        return "ci-cd"
    if ".claude" in parts:
        if "hooks" in parts:
            return "hooks"
        if "commands" in parts:
            return "commands"
        if "skills" in parts:
            return "skills"
        if "agents" in parts:
            return "agents"
        return "ai-config"
    if "docs" in parts or "documentation" in parts:
        return "documentation"

    # By extension
    ext_categories = {
        ".py": "python",
        ".js": "javascript",
        ".ts": "typescript",
        ".tsx": "react",
        ".jsx": "react",
        ".go": "golang",
        ".rs": "rust",
        ".sh": "shell",
        ".bash": "shell",
        ".yml": "configuration",
        ".yaml": "configuration",
        ".json": "configuration",
        ".toml": "configuration",
        ".md": "documentation",
        ".sql": "database",
        ".css": "styles",
        ".scss": "styles",
        ".html": "markup",
    }
    return ext_categories.get(ext, "other")


def analyze_diff(diff: str) -> dict:
    """Analyze a diff to understand what changed.

    Performs a single pass over the diff lines to calculate stats
    and detect patterns simultaneously.
    """
    additions = 0
    deletions = 0
    patterns = {
        "new_function": False,
        "new_class": False,
        "imports_changed": False,
        "config_changed": False,
        "tests_added": False,
        "error_handling": False,
        "comments_added": False,
    }

    for line in diff.split("\n"):
        # Skip diff metadata lines
        if line.startswith("+++") or line.startswith("---"):
            continue

        if line.startswith("+"):
            additions += 1
            content = line[1:].strip()

            # Detect patterns in added lines
            if content.startswith(("def ", "function ", "func ")):
                patterns["new_function"] = True
            elif content.startswith("class "):
                patterns["new_class"] = True
            elif content.startswith(("import ", "from ")):
                patterns["imports_changed"] = True
            elif "test" in content.lower() and any(x in content for x in ("def ", "it(", "describe(")):
                patterns["tests_added"] = True
            elif any(x in content for x in ("try:", "catch", "except")):
                patterns["error_handling"] = True
            elif content.startswith(("#", "//", "/*")):
                patterns["comments_added"] = True

        elif line.startswith("-"):
            deletions += 1

    return {
        "additions": additions,
        "deletions": deletions,
        "patterns": patterns,
    }


def infer_change_type(categories: set[str], patterns_by_file: dict[str, dict]) -> str:
    """Infer the type of change based on categories and patterns."""
    # Aggregate all patterns
    all_patterns: dict[str, bool] = {}
    for file_patterns in patterns_by_file.values():
        for key, value in file_patterns.items():
            if value:
                all_patterns[key] = True

    # Check for specific change types
    if "tests" in categories:
        if all_patterns.get("tests_added"):
            return "test"
        return "test-update"
    if "ci-cd" in categories or "hooks" in categories:
        return "ci"
    if "documentation" in categories and len(categories) == 1:
        return "docs"
    if "dependencies" in categories:
        return "deps"
    if all_patterns.get("new_function") or all_patterns.get("new_class"):
        return "feat"
    if "ai-config" in categories or "commands" in categories or "skills" in categories:
        return "feat"

    # If we can't determine from categories/patterns, default to chore
    # (stats-based refinement happens in generate_context after this)
    if all_patterns:
        return "feat"
    return "chore"


def generate_context(
    changed_files: list[str],
    base: str | None = None,
    head: str | None = None,
) -> dict:
    """Generate context document for changes.

    Fetches the full diff once and parses it to avoid N+1 git calls.
    """
    if not changed_files:
        return {
            "summary": "No changes detected",
            "files": [],
            "categories": {},
            "change_type": "none",
        }

    # Get full diff once and parse it
    full_diff = get_full_diff(base, head)
    file_diffs = parse_diff_by_file(full_diff)

    # Categorize files and analyze diffs
    categories: dict[str, list[str]] = {}
    file_analyses: dict[str, dict] = {}

    for filepath in changed_files:
        # Categorize
        category = categorize_file(filepath)
        if category not in categories:
            categories[category] = []
        categories[category].append(filepath)

        # Analyze diff (from parsed data, not a new git call)
        diff_content = file_diffs.get(filepath, "")
        file_analyses[filepath] = analyze_diff(diff_content)

    # Infer change type
    patterns_by_file = {f: a["patterns"] for f, a in file_analyses.items()}
    change_type = infer_change_type(set(categories.keys()), patterns_by_file)

    # Recalculate change type based on actual additions/deletions
    total_additions = sum(a["additions"] for a in file_analyses.values())
    total_deletions = sum(a["deletions"] for a in file_analyses.values())

    if change_type == "chore" and total_deletions > total_additions * 2:
        change_type = "refactor"
    elif change_type == "chore" and total_additions > 0:
        change_type = "feat"

    # Generate summary
    total_files = len(changed_files)
    summary = f"{total_files} file(s) changed (+{total_additions}/-{total_deletions})"

    return {
        "summary": summary,
        "files": changed_files,
        "categories": categories,
        "file_analyses": file_analyses,
        "change_type": change_type,
        "total_additions": total_additions,
        "total_deletions": total_deletions,
        "timestamp": datetime.now().isoformat(),
        "mode": "pr-diff" if base and head else "staged",
    }


def format_markdown(context: dict) -> str:
    """Format context as markdown."""
    mode_label = "PR Diff" if context.get("mode") == "pr-diff" else "Staged Changes"
    lines = [
        "# Commit Context",
        "",
        f"**Generated:** {context.get('timestamp', 'unknown')}",
        f"**Mode:** {mode_label}",
        f"**Change Type:** `{context.get('change_type', 'unknown')}`",
        "",
        "## Summary",
        "",
        context.get("summary", "No summary"),
        "",
    ]

    categories = context.get("categories", {})
    if categories:
        lines.extend(["## Changes by Category", ""])
        for cat, files in sorted(categories.items()):
            lines.append(f"### {cat.replace('-', ' ').title()}")
            for f in files:
                analysis = context.get("file_analyses", {}).get(f, {})
                adds = analysis.get("additions", 0)
                dels = analysis.get("deletions", 0)
                lines.append(f"- `{f}` (+{adds}/-{dels})")
            lines.append("")

    # Add pattern insights
    all_patterns = set()
    for analysis in context.get("file_analyses", {}).values():
        for pattern, found in analysis.get("patterns", {}).items():
            if found:
                all_patterns.add(pattern)

    if all_patterns:
        lines.extend(["## Detected Patterns", ""])
        pattern_descriptions = {
            "new_function": "New functions/methods added",
            "new_class": "New classes defined",
            "imports_changed": "Import statements modified",
            "tests_added": "Test cases added",
            "error_handling": "Error handling added/modified",
            "comments_added": "Comments/documentation added",
        }
        for pattern in sorted(all_patterns):
            desc = pattern_descriptions.get(pattern, pattern.replace("_", " ").title())
            lines.append(f"- {desc}")
        lines.append("")

    return "\n".join(lines)


def format_json(context: dict) -> str:
    """Format context as JSON for machine consumption."""
    # Remove non-serializable items and large data
    clean = {
        "summary": context.get("summary"),
        "change_type": context.get("change_type"),
        "files": context.get("files"),
        "categories": context.get("categories"),
        "total_additions": context.get("total_additions"),
        "total_deletions": context.get("total_deletions"),
        "timestamp": context.get("timestamp"),
        "mode": context.get("mode"),
    }
    return json.dumps(clean, indent=2)


def main():
    """Main entry point."""
    parser = argparse.ArgumentParser(
        description="Generate context documentation for code changes"
    )
    parser.add_argument(
        "--base",
        help="Base ref for comparison (e.g., origin/main). If not provided, uses staged changes.",
    )
    parser.add_argument(
        "--head",
        help="Head ref for comparison (e.g., HEAD). Required if --base is provided.",
    )
    parser.add_argument(
        "--stdout",
        action="store_true",
        help="Output markdown to stdout only (don't write files)",
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Output JSON instead of markdown (implies --stdout)",
    )
    parser.add_argument(
        "--output-dir",
        default=".claude/artifacts",
        help="Directory to write output files (default: .claude/artifacts)",
    )

    args = parser.parse_args()

    # Validate args
    if args.base and not args.head:
        parser.error("--head is required when --base is provided")
    if args.head and not args.base:
        parser.error("--base is required when --head is provided")

    # Read stdin (hook input) - ignored for now but could be used
    try:
        if not sys.stdin.isatty():
            _ = sys.stdin.read()
    except Exception:
        pass

    # Get changed files
    changed_files = get_changed_files(args.base, args.head)

    if not changed_files:
        if args.json:
            print(json.dumps({"summary": "No changes", "files": [], "change_type": "none"}))
        else:
            print("No changes to document")
        sys.exit(0)

    # Generate context
    context = generate_context(changed_files, args.base, args.head)

    # Output
    if args.json:
        print(format_json(context))
    elif args.stdout:
        print(format_markdown(context))
    else:
        # Write files and print summary
        artifacts_dir = Path(args.output_dir)
        artifacts_dir.mkdir(parents=True, exist_ok=True)

        # Save markdown context
        md_content = format_markdown(context)
        md_path = artifacts_dir / "commit-context.md"
        md_path.write_text(md_content)

        # Save JSON context for machine consumption
        json_content = format_json(context)
        json_path = artifacts_dir / "commit-context.json"
        json_path.write_text(json_content)

        # Print summary to stdout
        mode = "PR diff" if args.base else "staged changes"
        print(f"Commit Context Generated ({mode})")
        print("=" * 40)
        print(f"Type: {context['change_type']}")
        print(f"Files: {len(changed_files)}")
        print(f"Changes: +{context['total_additions']}/-{context['total_deletions']}")
        print("")
        print("Categories:")
        for cat, files in sorted(context.get("categories", {}).items()):
            print(f"  - {cat}: {len(files)} file(s)")
        print("")
        print(f"Context saved to: {md_path}")
        print("=" * 40)

    sys.exit(0)


if __name__ == "__main__":
    main()
