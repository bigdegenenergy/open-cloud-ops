"""
Progress Tracking Utilities
===========================

Functions for tracking and displaying progress of the autonomous coding agent.
"""

import json
from pathlib import Path


def count_passing_tests(project_dir: Path) -> tuple[int, int]:
    """
    Count passing and total tests in feature_list.json.

    Args:
        project_dir: Directory containing feature_list.json

    Returns:
        (passing_count, total_count)
    """
    tests_file = project_dir / "feature_list.json"

    if not tests_file.exists():
        # File doesn't exist yet - this is normal on first run
        return 0, 0

    # Prevent OOM attacks from hallucinated oversized JSON
    # Limit to 10MB to catch runaway LLM output
    MAX_SIZE_MB = 10
    try:
        file_size_mb = tests_file.stat().st_size / (1024 * 1024)
        if file_size_mb > MAX_SIZE_MB:
            print(f"⚠️  Warning: feature_list.json is too large ({file_size_mb:.1f}MB > {MAX_SIZE_MB}MB limit)")
            return 0, 0
    except OSError:
        pass  # If we can't stat, let json.load handle the error

    try:
        with open(tests_file, "r") as f:
            data = json.load(f)

        # Validate structure: must be a list
        if not isinstance(data, list):
            print(f"⚠️  Warning: feature_list.json must be an array, got {type(data).__name__}")
            return 0, 0

        total = len(data)
        passing = sum(1 for test in data if isinstance(test, dict) and test.get("passes", False))

        return passing, total
    except json.JSONDecodeError as e:
        # Invalid JSON in feature_list.json
        print(f"⚠️  Warning: feature_list.json is not valid JSON: {e}")
        return 0, 0
    except IOError as e:
        # File permission or I/O error
        print(f"⚠️  Warning: Could not read feature_list.json: {e}")
        return 0, 0
    except Exception as e:
        # Catch any other unexpected errors
        print(f"⚠️  Warning: Unexpected error reading feature_list.json: {e}")
        return 0, 0


def print_session_header(session_num: int, is_initializer: bool) -> None:
    """Print a formatted header for the session."""
    session_type = "INITIALIZER" if is_initializer else "CODING AGENT"

    print("\n" + "=" * 70)
    print(f"  SESSION {session_num}: {session_type}")
    print("=" * 70)
    print()


def print_progress_summary(project_dir: Path) -> None:
    """Print a summary of current progress."""
    passing, total = count_passing_tests(project_dir)

    if total > 0:
        percentage = (passing / total) * 100
        print(f"\nProgress: {passing}/{total} tests passing ({percentage:.1f}%)")
    else:
        print("\nProgress: feature_list.json not yet created")
