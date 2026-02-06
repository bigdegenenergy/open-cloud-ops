"""Workflow engine stub for Claude Code command integration.

This module provides the WorkflowEngine class referenced by the
/workflow and /workflow-approve slash commands. It manages typed
pipelines with approval gates.

Full implementation is planned for Phase 4 (Enterprise).
"""

import json
import os
from pathlib import Path


class WorkflowEngine:
    """Manages typed workflow pipelines with approval gates."""

    def __init__(self, base_dir=None):
        self.base_dir = Path(base_dir or os.getcwd()) / ".claude" / "workflows"
        self.definitions_dir = self.base_dir / "definitions"
        self.state_dir = Path(os.getcwd()) / ".claude" / "artifacts" / "workflow-state"

    def load_workflow(self, name):
        """Load a workflow definition by name."""
        path = self.definitions_dir / f"{name}.yaml"
        if not path.exists():
            raise FileNotFoundError(
                f"Workflow '{name}' not found. "
                f"Available workflows: {self._list_workflows()}"
            )
        # Stub: return the raw YAML path for now
        return {"name": name, "path": str(path)}

    def start(self, workflow, variables=None):
        """Start a workflow execution."""
        self.state_dir.mkdir(parents=True, exist_ok=True)
        state = {
            "workflow": workflow["name"],
            "status": "running",
            "variables": variables or {},
            "current_step": 0,
        }
        state_path = self.state_dir / f"{workflow['name']}.state.json"
        state_path.write_text(json.dumps(state, indent=2))
        return state

    def approve(self, workflow_id, approved_by=""):
        """Approve a pending workflow gate."""
        approval_path = self.state_dir / f"{workflow_id}.approval.json"
        if not approval_path.exists():
            raise FileNotFoundError(f"No pending approval for workflow '{workflow_id}'")
        approval = json.loads(approval_path.read_text())
        approval["status"] = "approved"
        approval["approved_by"] = approved_by
        approval_path.write_text(json.dumps(approval, indent=2))
        return approval

    def _list_workflows(self):
        """List available workflow definitions."""
        if not self.definitions_dir.exists():
            return []
        return [p.stem for p in self.definitions_dir.glob("*.yaml")]
