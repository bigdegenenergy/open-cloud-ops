"""
Agent Session Logic
===================

Core agent interaction functions for running autonomous coding sessions.
"""

import asyncio
import json
from pathlib import Path
from typing import Optional

from claude_code_sdk import ClaudeSDKClient

from client import create_client
from progress import print_session_header, print_progress_summary, count_passing_tests
from prompts import get_initializer_prompt, get_coding_prompt, copy_spec_to_project


# Configuration
AUTO_CONTINUE_DELAY_SECONDS = 3


def validate_feature_list_json(project_dir: Path) -> bool:
    """
    Validate that feature_list.json has the expected schema.
    
    Expected format: Array of objects with required 'passes' field.
    
    Args:
        project_dir: Directory containing feature_list.json
        
    Returns:
        True if valid, False otherwise
    """
    tests_file = project_dir / "feature_list.json"
    
    if not tests_file.exists():
        return True  # File doesn't exist yet - no validation needed
    
    try:
        with open(tests_file, "r") as f:
            data = json.load(f)
        
        # Must be a list
        if not isinstance(data, list):
            print(f"❌ feature_list.json validation failed: root must be an array, got {type(data).__name__}")
            return False
        
        # Each item must be a dict with 'passes' field
        for i, item in enumerate(data):
            if not isinstance(item, dict):
                print(f"❌ feature_list.json validation failed: item {i} must be object, got {type(item).__name__}")
                return False
            if "passes" not in item:
                print(f"❌ feature_list.json validation failed: item {i} missing required 'passes' field")
                return False
            if not isinstance(item["passes"], bool):
                print(f"❌ feature_list.json validation failed: item {i} 'passes' must be boolean")
                return False
        
        return True
        
    except json.JSONDecodeError as e:
        print(f"❌ feature_list.json validation failed: invalid JSON: {e}")
        return False
    except Exception as e:
        print(f"❌ feature_list.json validation failed: {e}")
        return False


async def run_agent_session(
    client: ClaudeSDKClient,
    message: str,
    project_dir: Path,
) -> tuple[str, str]:
    """
    Run a single agent session using Claude Agent SDK.

    Args:
        client: Claude SDK client
        message: The prompt to send
        project_dir: Project directory path

    Returns:
        (status, response_text) where status is:
        - "continue" if agent should continue working
        - "error" if an error occurred
    """
    print("Sending prompt to Claude Agent SDK...\n")

    try:
        # Send the query
        await client.query(message)

        # Collect response text and show tool use
        response_text = ""
        async for msg in client.receive_response():
            msg_type = type(msg).__name__

            # Handle AssistantMessage (text and tool use)
            if msg_type == "AssistantMessage" and hasattr(msg, "content"):
                for block in msg.content:
                    block_type = type(block).__name__

                    if block_type == "TextBlock" and hasattr(block, "text"):
                        response_text += block.text
                        print(block.text, end="", flush=True)
                    elif block_type == "ToolUseBlock" and hasattr(block, "name"):
                        print(f"\n[Tool: {block.name}]", flush=True)
                        if hasattr(block, "input"):
                            input_str = str(block.input)
                            if len(input_str) > 200:
                                print(f"   Input: {input_str[:200]}...", flush=True)
                            else:
                                print(f"   Input: {input_str}", flush=True)

            # Handle UserMessage (tool results)
            elif msg_type == "UserMessage" and hasattr(msg, "content"):
                for block in msg.content:
                    block_type = type(block).__name__

                    if block_type == "ToolResultBlock":
                        result_content = getattr(block, "content", "")
                        is_error = getattr(block, "is_error", False)

                        # Check if command was blocked by security hook
                        if "blocked" in str(result_content).lower():
                            print(f"   [BLOCKED] {result_content}", flush=True)
                        elif is_error:
                            # Show errors (truncated)
                            error_str = str(result_content)[:500]
                            print(f"   [Error] {error_str}", flush=True)
                        else:
                            # Tool succeeded - just show brief confirmation
                            print("   [Done]", flush=True)

        print("\n" + "-" * 70 + "\n")
        return "continue", response_text

    except Exception as e:
        print(f"Error during agent session: {e}")
        return "error", str(e)


async def run_autonomous_agent(
    project_dir: Path,
    model: str,
    max_iterations: Optional[int] = None,
) -> None:
    """
    Run the autonomous agent loop.

    Args:
        project_dir: Directory for the project
        model: Claude model to use
        max_iterations: Maximum number of iterations (default: 20 to prevent runaway loops)
    """
    # Set sensible default to prevent infinite loops and runaway costs
    if max_iterations is None:
        max_iterations = 20

    print("\n" + "=" * 70)
    print("  AUTONOMOUS CODING AGENT DEMO")
    print("=" * 70)
    print(f"\nProject directory: {project_dir}")
    print(f"Model: {model}")
    print(f"Max iterations: {max_iterations}")
    print()

    # Create project directory
    project_dir.mkdir(parents=True, exist_ok=True)

    # Check if this is a fresh start or continuation
    tests_file = project_dir / "feature_list.json"
    is_first_run = not tests_file.exists()

    # If feature_list.json exists, validate it before proceeding
    # If validation fails, treat as invalid and re-run initializer
    if tests_file.exists() and not is_first_run:
        if not validate_feature_list_json(project_dir):
            print("⚠️  Existing feature_list.json is invalid - re-running initializer")
            is_first_run = True  # Force re-initialization
            # Don't delete the file - let initializer overwrite it
        else:
            print("✅ Existing feature_list.json validated")

    if is_first_run:
        print("Fresh start - will use initializer agent")
        print()
        print("=" * 70)
        print("  NOTE: First session takes 10-20+ minutes!")
        print("  The agent is generating 200 detailed test cases.")
        print("  This may appear to hang - it's working. Watch for [Tool: ...] output.")
        print("=" * 70)
        print()
        # Copy the app spec into the project directory for the agent to read
        copy_spec_to_project(project_dir)
    else:
        print("Continuing existing project")
        print_progress_summary(project_dir)

    # Main loop
    iteration = 0

    while True:
        iteration += 1

        # Check max iterations
        if iteration > max_iterations:
            print(f"\nReached max iterations ({max_iterations})")
            print("To continue, run the script again with a higher --max-iterations value")
            break

        # Check for completion: all tests passing
        passing, total = count_passing_tests(project_dir)
        if total > 0 and passing == total:
            print("\n" + "=" * 70)
            print("  🎉 ALL TESTS PASSING - PROJECT COMPLETE!")
            print("=" * 70)
            print(f"\nCompleted {total}/{total} features")
            print(f"Project ready at: {project_dir.resolve()}")
            break

        # Print session header
        print_session_header(iteration, is_first_run)

        # Create client (fresh context)
        client = create_client(project_dir, model)

        # Choose prompt based on session type
        if is_first_run:
            prompt = get_initializer_prompt()
        else:
            # Before switching to coding phase, ensure feature_list.json exists and is valid
            if not (project_dir / "feature_list.json").exists():
                print("❌ ERROR: feature_list.json does not exist!")
                print("Cannot proceed to coding phase without initializer output")
                print("Re-running initializer...")
                is_first_run = True
                prompt = get_initializer_prompt()
            elif not validate_feature_list_json(project_dir):
                print("❌ ERROR: feature_list.json is invalid or corrupted!")
                print("Cannot proceed to coding phase with invalid feature list")
                print("Re-running initializer...")
                is_first_run = True
                prompt = get_initializer_prompt()
            else:
                prompt = get_coding_prompt()

        # Run session with async context manager
        async with client:
            status, response = await run_agent_session(client, prompt, project_dir)

        # Mark initializer as complete only after successful run
        # (If initializer failed, we'll retry it in the next iteration)
        if is_first_run and (project_dir / "feature_list.json").exists():
            # Validate the feature_list.json schema before accepting it
            if validate_feature_list_json(project_dir):
                print("✅ Initializer session complete - feature_list.json created and validated")
                is_first_run = False  # Only use initializer once
            else:
                print("⚠️  feature_list.json was created but failed schema validation")
                print("Retrying initializer in next session...")
                # Don't mark as complete - will retry in next iteration

        # Handle status
        if status == "continue":
            print(f"\nAgent will auto-continue in {AUTO_CONTINUE_DELAY_SECONDS}s...")
            print_progress_summary(project_dir)
            await asyncio.sleep(AUTO_CONTINUE_DELAY_SECONDS)  # Use await, not time.sleep

        elif status == "error":
            print("\nSession encountered an error")
            if is_first_run:
                print("Retrying initializer in next session...")
            else:
                print("Will retry with a fresh session...")
            await asyncio.sleep(AUTO_CONTINUE_DELAY_SECONDS)

        # Small delay between sessions
        if iteration < max_iterations:
            print("\nPreparing next session...\n")
            await asyncio.sleep(1)

    # Final summary
    print("\n" + "=" * 70)
    print("  SESSION COMPLETE")
    print("=" * 70)
    print(f"\nProject directory: {project_dir}")
    print_progress_summary(project_dir)

    # Print instructions for running the generated application
    print("\n" + "-" * 70)
    print("  TO RUN THE GENERATED APPLICATION:")
    print("-" * 70)
    print(f"\n  cd {project_dir.resolve()}")
    print("  ./init.sh           # Run the setup script")
    print("  # Or manually:")
    print("  npm install && npm run dev")
    print("\n  Then open http://localhost:3000 (or check init.sh for the URL)")
    print("-" * 70)

    print("\nDone!")
