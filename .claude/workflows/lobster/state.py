#!/usr/bin/env python3
"""
Lobster Workflow Engine - State Persistence

Manages workflow state with file-based persistence in .claude/artifacts/workflow-state/.
Supports concurrent access and state recovery.
"""

import json
import os
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional

# Cross-platform file locking
try:
    import portalocker

    HAS_PORTALOCKER = True
except ImportError:
    # Fallback to platform-specific locking
    import sys

    if sys.platform == "win32":
        import msvcrt
    else:
        import fcntl
    HAS_PORTALOCKER = False

from .types import (
    WorkflowState,
    WorkflowStatus,
    StepResult,
    StepStatus,
    ApprovalRequest,
)


class StateManager:
    """Manages workflow state persistence."""

    def __init__(self, artifacts_dir: Optional[Path] = None):
        """
        Initialize the state manager.

        Args:
            artifacts_dir: Directory for state files. Defaults to
                          .claude/artifacts/workflow-state/
        """
        if artifacts_dir is None:
            # Find .claude directory relative to this file
            current_dir = Path(__file__).parent.parent.parent
            artifacts_dir = current_dir / "artifacts" / "workflow-state"

        self.artifacts_dir = Path(artifacts_dir)
        self.artifacts_dir.mkdir(parents=True, exist_ok=True)

    def _acquire_lock(self, file_obj, exclusive: bool = True):
        """Acquire a file lock in a cross-platform way."""
        if HAS_PORTALOCKER:
            # Use portalocker for cross-platform locking
            flags = portalocker.LOCK_EX if exclusive else portalocker.LOCK_SH
            portalocker.lock(file_obj, flags)
        elif os.name == "nt":  # Windows
            # Use msvcrt for Windows
            import msvcrt

            # Lock the first byte of the file
            file_obj.seek(0)
            msvcrt.locking(file_obj.fileno(), msvcrt.LK_LOCK, 1)
        else:  # Unix-like systems
            # Use fcntl for Unix
            import fcntl

            lock_type = fcntl.LOCK_EX if exclusive else fcntl.LOCK_SH
            fcntl.flock(file_obj.fileno(), lock_type)

    def _release_lock(self, file_obj):
        """Release a file lock in a cross-platform way."""
        if HAS_PORTALOCKER:
            portalocker.unlock(file_obj)
        elif os.name == "nt":  # Windows
            import msvcrt

            # Unlock the first byte of the file
            file_obj.seek(0)
            msvcrt.locking(file_obj.fileno(), msvcrt.LK_UNLCK, 1)
        else:  # Unix-like systems
            import fcntl

            fcntl.flock(file_obj.fileno(), fcntl.LOCK_UN)

    def _sanitize_workflow_id(self, workflow_id: str) -> str:
        """
        Sanitize workflow_id to prevent path traversal attacks.

        Args:
            workflow_id: The workflow ID to sanitize

        Returns:
            Sanitized workflow ID containing only safe characters

        Raises:
            ValueError: If workflow_id contains invalid characters
        """
        # Use basename to strip any directory components
        safe_id = os.path.basename(workflow_id)

        # Validate that it contains only alphanumeric, hyphens, and underscores
        import re

        if not re.match(r"^[\w\-]+$", safe_id):
            raise ValueError(
                f"Invalid workflow_id: {workflow_id}. "
                "Only alphanumeric characters, hyphens, and underscores are allowed."
            )

        return safe_id

    def _state_file(self, workflow_id: str) -> Path:
        """Get path to state file for a workflow."""
        safe_id = self._sanitize_workflow_id(workflow_id)
        return self.artifacts_dir / f"{safe_id}.json"

    def _approval_file(self, workflow_id: str) -> Path:
        """Get path to approval file for a workflow."""
        safe_id = self._sanitize_workflow_id(workflow_id)
        return self.artifacts_dir / f"{safe_id}.approval.json"

    def _lock_file(self, workflow_id: str) -> Path:
        """Get path to lock file for concurrent access."""
        safe_id = self._sanitize_workflow_id(workflow_id)
        return self.artifacts_dir / f"{safe_id}.lock"

    def _serialize_datetime(self, dt: Optional[datetime]) -> Optional[str]:
        """Convert datetime to ISO format string."""
        if dt is None:
            return None
        return dt.isoformat()

    def _deserialize_datetime(self, s: Optional[str]) -> Optional[datetime]:
        """
        Convert ISO format string to timezone-aware UTC datetime.

        Handles both naive and aware datetimes for backward compatibility.
        Naive datetimes are assumed to be UTC.
        """
        if s is None:
            return None
        dt = datetime.fromisoformat(s)
        # If the datetime is naive (no timezone info), assume it's UTC
        if dt.tzinfo is None:
            dt = dt.replace(tzinfo=timezone.utc)
        return dt

    def save_state(self, state: WorkflowState) -> None:
        """
        Save workflow state to disk.

        Args:
            state: The workflow state to save
        """
        state_file = self._state_file(state.workflow_id)
        lock_file = self._lock_file(state.workflow_id)

        # Ensure updated_at is current
        state.updated_at = datetime.now(timezone.utc)

        # Serialize state to JSON
        data = {
            "workflow_name": state.workflow_name,
            "workflow_id": state.workflow_id,
            "status": state.status.value,
            "current_step": state.current_step,
            "step_results": [
                {
                    "step_name": r.step_name,
                    "status": r.status.value,
                    "output": r.output,
                    "error": r.error,
                    "started_at": self._serialize_datetime(r.started_at),
                    "completed_at": self._serialize_datetime(r.completed_at),
                    "duration_seconds": r.duration_seconds,
                    "retry_attempts": r.retry_attempts,
                }
                for r in state.step_results
            ],
            "pending_approval": state.pending_approval,
            "variables": state.variables,
            "started_at": self._serialize_datetime(state.started_at),
            "updated_at": self._serialize_datetime(state.updated_at),
            "completed_at": self._serialize_datetime(state.completed_at),
            "error": state.error,
        }

        # Write with file locking for concurrent access
        with open(lock_file, "a") as lf:
            self._acquire_lock(lf, exclusive=True)
            try:
                with open(state_file, "w") as f:
                    json.dump(data, f, indent=2)
            finally:
                self._release_lock(lf)

    def load_state(self, workflow_id: str) -> Optional[WorkflowState]:
        """
        Load workflow state from disk.

        Args:
            workflow_id: ID of the workflow

        Returns:
            The workflow state, or None if not found
        """
        state_file = self._state_file(workflow_id)

        if not state_file.exists():
            return None

        lock_file = self._lock_file(workflow_id)

        with open(lock_file, "a") as lf:
            self._acquire_lock(lf, exclusive=False)
            try:
                with open(state_file) as f:
                    data = json.load(f)
            finally:
                self._release_lock(lf)

        # Deserialize step results
        step_results = [
            StepResult(
                step_name=r["step_name"],
                status=StepStatus(r["status"]),
                output=r.get("output"),
                error=r.get("error"),
                started_at=self._deserialize_datetime(r.get("started_at")),
                completed_at=self._deserialize_datetime(r.get("completed_at")),
                duration_seconds=r.get("duration_seconds", 0.0),
                retry_attempts=r.get("retry_attempts", 0),
            )
            for r in data.get("step_results", [])
        ]

        return WorkflowState(
            workflow_name=data["workflow_name"],
            workflow_id=data["workflow_id"],
            status=WorkflowStatus(data["status"]),
            current_step=data.get("current_step"),
            step_results=step_results,
            pending_approval=data.get("pending_approval"),
            variables=data.get("variables", {}),
            started_at=self._deserialize_datetime(data.get("started_at")),
            updated_at=self._deserialize_datetime(data.get("updated_at")),
            completed_at=self._deserialize_datetime(data.get("completed_at")),
            error=data.get("error"),
        )

    def delete_state(self, workflow_id: str) -> bool:
        """
        Delete workflow state from disk.

        Args:
            workflow_id: ID of the workflow

        Returns:
            True if deleted, False if not found
        """
        state_file = self._state_file(workflow_id)
        approval_file = self._approval_file(workflow_id)
        lock_file = self._lock_file(workflow_id)

        deleted = False

        for f in [state_file, approval_file, lock_file]:
            if f.exists():
                f.unlink()
                deleted = True

        return deleted

    def list_workflows(self) -> list[str]:
        """
        List all workflow IDs with saved state.

        Returns:
            List of workflow IDs
        """
        workflows = []
        for f in self.artifacts_dir.glob("*.json"):
            if not f.name.endswith(".approval.json"):
                workflow_id = f.stem
                workflows.append(workflow_id)
        return workflows

    def list_active_workflows(self) -> list[WorkflowState]:
        """
        List all active (running or paused) workflows.

        Returns:
            List of active workflow states
        """
        active = []
        for workflow_id in self.list_workflows():
            state = self.load_state(workflow_id)
            if state and state.status in [
                WorkflowStatus.RUNNING,
                WorkflowStatus.PAUSED,
            ]:
                active.append(state)
        return active

    def save_approval_request(self, request: ApprovalRequest) -> None:
        """
        Save an approval request.

        Args:
            request: The approval request to save
        """
        approval_file = self._approval_file(request.workflow_id)
        lock_file = self._lock_file(request.workflow_id)

        data = {
            "workflow_id": request.workflow_id,
            "step_name": request.step_name,
            "gate": {
                "gate_type": request.gate.gate_type.value,
                "message": request.gate.message,
                "timeout_seconds": request.gate.timeout_seconds,
                "condition": request.gate.condition,
                "fallback": request.gate.fallback,
                "approvers": request.gate.approvers,
                "notify": request.gate.notify,
            },
            "requested_at": self._serialize_datetime(request.requested_at),
            "expires_at": self._serialize_datetime(request.expires_at),
            "approved_by": request.approved_by,
            "approved_at": self._serialize_datetime(request.approved_at),
            "rejected_by": request.rejected_by,
            "rejected_at": self._serialize_datetime(request.rejected_at),
            "rejection_reason": request.rejection_reason,
        }

        # Write with file locking for concurrent access
        with open(lock_file, "a") as lf:
            self._acquire_lock(lf, exclusive=True)
            try:
                with open(approval_file, "w") as f:
                    json.dump(data, f, indent=2)
            finally:
                self._release_lock(lf)

    def load_approval_request(self, workflow_id: str) -> Optional[ApprovalRequest]:
        """
        Load an approval request.

        Args:
            workflow_id: ID of the workflow

        Returns:
            The approval request, or None if not found
        """
        from .types import ApprovalGate, GateType

        approval_file = self._approval_file(workflow_id)

        if not approval_file.exists():
            return None

        with open(approval_file) as f:
            data = json.load(f)

        gate_data = data["gate"]
        gate = ApprovalGate(
            gate_type=GateType(gate_data["gate_type"]),
            message=gate_data["message"],
            timeout_seconds=gate_data.get("timeout_seconds"),
            condition=gate_data.get("condition"),
            fallback=gate_data.get("fallback", "fail"),
            approvers=gate_data.get("approvers", []),
            notify=gate_data.get("notify", True),
        )

        return ApprovalRequest(
            workflow_id=data["workflow_id"],
            step_name=data["step_name"],
            gate=gate,
            requested_at=self._deserialize_datetime(data["requested_at"]),
            expires_at=self._deserialize_datetime(data.get("expires_at")),
            approved_by=data.get("approved_by"),
            approved_at=self._deserialize_datetime(data.get("approved_at")),
            rejected_by=data.get("rejected_by"),
            rejected_at=self._deserialize_datetime(data.get("rejected_at")),
            rejection_reason=data.get("rejection_reason"),
        )

    def list_pending_approvals(self) -> list[ApprovalRequest]:
        """
        List all pending approval requests.

        Returns:
            List of pending approval requests
        """
        pending = []
        for f in self.artifacts_dir.glob("*.approval.json"):
            workflow_id = f.stem.replace(".approval", "")
            request = self.load_approval_request(workflow_id)
            if request and request.is_pending and not request.is_expired:
                pending.append(request)
        return pending
