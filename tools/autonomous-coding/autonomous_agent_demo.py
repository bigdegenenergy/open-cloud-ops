#!/usr/bin/env python3
"""
Autonomous Coding Agent Demo
============================

Entry point for the autonomous coding agent system.
Builds complete applications over multiple sessions using Claude Agent SDK.

Usage:
    python autonomous_agent_demo.py --project-dir ./my_app
    python autonomous_agent_demo.py --project-dir ./my_app --max-iterations 3
    python autonomous_agent_demo.py --project-dir ./my_app --model claude-opus-4.6
"""

import asyncio
import argparse
import sys
from pathlib import Path

from agent import run_autonomous_agent


def main():
    parser = argparse.ArgumentParser(
        description="Autonomous Coding Agent Demo",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Start a fresh project
  python autonomous_agent_demo.py --project-dir ./my_app

  # Continue existing project
  python autonomous_agent_demo.py --project-dir ./my_app

  # Test with limited iterations
  python autonomous_agent_demo.py --project-dir ./my_app --max-iterations 3

  # Use a specific model
  python autonomous_agent_demo.py --project-dir ./my_app --model claude-opus-4.6
        """,
    )

    parser.add_argument(
        "--project-dir",
        type=Path,
        default=Path("./autonomous_demo_project"),
        help="Directory for the project (default: ./autonomous_demo_project)",
    )

    parser.add_argument(
        "--max-iterations",
        type=int,
        default=20,
        help="Maximum number of iterations (default: 20 to prevent runaway costs)",
    )

    parser.add_argument(
        "--model",
        type=str,
        default="claude-sonnet-4.5-20250929",
        help="Claude model to use (default: claude-sonnet-4.5-20250929)",
    )

    args = parser.parse_args()

    # Run the agent
    try:
        asyncio.run(
            run_autonomous_agent(
                project_dir=args.project_dir,
                model=args.model,
                max_iterations=args.max_iterations,
            )
        )
    except KeyboardInterrupt:
        print("\n\n" + "=" * 70)
        print("  PAUSED")
        print("=" * 70)
        print(f"\nProject saved to: {args.project_dir.resolve()}")
        print("\nTo resume, run:")
        print(f"  python autonomous_agent_demo.py --project-dir {args.project_dir}")
        print("\n" + "=" * 70)
        sys.exit(0)
    except Exception as e:
        print(f"\nError: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
