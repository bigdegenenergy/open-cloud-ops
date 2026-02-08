"""
Prompt Loading Utilities
========================

Functions for loading prompt templates from the prompts directory.
"""

import os
import shutil
from pathlib import Path


PROMPTS_DIR = Path(__file__).parent / "prompts"


def load_prompt(name: str) -> str:
    """Load a prompt template from the prompts directory.
    
    Uses os.path.basename to prevent directory traversal attacks
    (e.g., load_prompt("../../../etc/passwd")).
    """
    # Prevent directory traversal by using only the filename
    safe_name = os.path.basename(name)
    prompt_path = PROMPTS_DIR / f"{safe_name}.md"
    return prompt_path.read_text()


def get_initializer_prompt() -> str:
    """Load the initializer prompt."""
    return load_prompt("initializer_prompt")


def get_coding_prompt() -> str:
    """Load the coding agent prompt."""
    return load_prompt("coding_prompt")


def copy_spec_to_project(project_dir: Path) -> None:
    """Copy the app spec file into the project directory for the agent to read."""
    spec_source = PROMPTS_DIR / "app_spec.txt"
    spec_dest = project_dir / "app_spec.txt"
    if not spec_dest.exists():
        shutil.copy(spec_source, spec_dest)
        print("Copied app_spec.txt to project directory")
