#!/usr/bin/env python3
"""
Lobster Workflow Engine - Type Definitions

Typed workflow schemas for deterministic, multi-step pipelines
with approval gates and state persistence.
"""

from dataclasses import dataclass, field
from datetime import datetime
from enum import Enum
from typing import Any, Optional


class StepType(Enum):
    """Type of workflow step."""

    COMMAND = "command"  # Execute a slash command (e.g., /plan, /qa)
    AGENT = "agent"  # Invoke a subagent (e.g., @code-reviewer)
    SHELL = "shell"  # Run a shell command
    PARALLEL = "parallel"  # Run multiple steps in parallel


class GateType(Enum):
    """Type of approval gate."""

    MANUAL = "manual"  # Requires explicit user approval
    TIMEOUT = "timeout"  # Auto-approves after duration
    CONDITIONAL = "conditional"  # Approves based on step output


class StepStatus(Enum):
    """Status of a workflow step."""

    PENDING = "pending"
    RUNNING = "running"
    WAITING_APPROVAL = "waiting_approval"
    APPROVED = "approved"
    REJECTED = "rejected"
    COMPLETED = "completed"
    FAILED = "failed"
    SKIPPED = "skipped"


class WorkflowStatus(Enum):
    """Status of the overall workflow."""

    NOT_STARTED = "not_started"
    RUNNING = "running"
    PAUSED = "paused"  # Waiting for approval
    COMPLETED = "completed"
    FAILED = "failed"
    CANCELLED = "cancelled"


@dataclass
class ApprovalGate:
    """
    An approval gate that pauses workflow execution.

    Attributes:
        gate_type: Type of gate (manual, timeout, conditional)
        message: Message to display when waiting for approval
        timeout_seconds: For timeout gates, auto-approve after this duration
        condition: For conditional gates, expression to evaluate
        fallback: What to do if condition fails ('continue', 'fail', 'skip')
        approvers: Optional list of users who can approve (empty = anyone)
        notify: Whether to send notification when gate is reached
    """

    gate_type: GateType = GateType.MANUAL
    message: str = "Approval required to continue"
    timeout_seconds: Optional[int] = None
    condition: Optional[str] = None
    fallback: str = "fail"
    approvers: list[str] = field(default_factory=list)
    notify: bool = True


@dataclass
class WorkflowStep:
    """
    A single step in a workflow.

    Attributes:
        name: Unique identifier for the step
        step_type: Type of step (command, agent, shell, parallel)
        target: The command, agent, or shell command to execute
        inputs: Input values for the step
        outputs: Names of outputs to capture
        gate: Optional approval gate after this step
        timeout_seconds: Maximum time for step execution
        retry_count: Number of retries on failure
        continue_on_failure: Whether to continue workflow if step fails
        depends_on: List of step names this step depends on
    """

    name: str
    step_type: StepType
    target: str
    inputs: dict[str, Any] = field(default_factory=dict)
    outputs: list[str] = field(default_factory=list)
    gate: Optional[ApprovalGate] = None
    timeout_seconds: int = 600  # 10 minutes default
    retry_count: int = 0
    continue_on_failure: bool = False
    depends_on: list[str] = field(default_factory=list)


@dataclass
class StepResult:
    """
    Result of executing a workflow step.

    Attributes:
        step_name: Name of the step
        status: Final status of the step
        output: Output produced by the step
        error: Error message if failed
        started_at: When the step started
        completed_at: When the step completed
        duration_seconds: How long the step took
        retry_attempts: Number of retries attempted
    """

    step_name: str
    status: StepStatus
    output: Any = None
    error: Optional[str] = None
    started_at: Optional[datetime] = None
    completed_at: Optional[datetime] = None
    duration_seconds: float = 0.0
    retry_attempts: int = 0


@dataclass
class WorkflowDefinition:
    """
    Complete workflow definition.

    Attributes:
        name: Unique name for the workflow
        description: Human-readable description
        version: Workflow version (for tracking changes)
        steps: Ordered list of steps to execute
        on_failure: Action when workflow fails ('notify', 'rollback', 'continue')
        on_success: Action when workflow completes ('notify', 'cleanup')
        timeout_seconds: Maximum total workflow duration
        metadata: Additional workflow metadata
    """

    name: str
    description: str = ""
    version: str = "1.0.0"
    steps: list[WorkflowStep] = field(default_factory=list)
    on_failure: str = "notify"
    on_success: str = "notify"
    timeout_seconds: int = 3600  # 1 hour default
    metadata: dict[str, Any] = field(default_factory=dict)


@dataclass
class WorkflowState:
    """
    Current state of a running workflow.

    Attributes:
        workflow_name: Name of the workflow being executed
        workflow_id: Unique ID for this execution
        status: Current workflow status
        current_step: Index of current step (or None if not started)
        step_results: Results of completed steps
        pending_approval: Step waiting for approval (if any)
        variables: Variables passed between steps
        started_at: When the workflow started
        updated_at: Last state update time
        completed_at: When the workflow completed (if done)
        error: Error message if failed
    """

    workflow_name: str
    workflow_id: str
    status: WorkflowStatus = WorkflowStatus.NOT_STARTED
    current_step: Optional[int] = None
    step_results: list[StepResult] = field(default_factory=list)
    pending_approval: Optional[str] = None
    variables: dict[str, Any] = field(default_factory=dict)
    started_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None
    completed_at: Optional[datetime] = None
    error: Optional[str] = None


@dataclass
class ApprovalRequest:
    """
    A request for workflow approval.

    Attributes:
        workflow_id: ID of the workflow
        step_name: Name of the step requiring approval
        gate: The approval gate configuration
        requested_at: When the approval was requested
        expires_at: When the request expires (for timeout gates)
        approved_by: Who approved (if approved)
        approved_at: When it was approved
        rejected_by: Who rejected (if rejected)
        rejected_at: When it was rejected
        rejection_reason: Why it was rejected
    """

    workflow_id: str
    step_name: str
    gate: ApprovalGate
    requested_at: datetime
    expires_at: Optional[datetime] = None
    approved_by: Optional[str] = None
    approved_at: Optional[datetime] = None
    rejected_by: Optional[str] = None
    rejected_at: Optional[datetime] = None
    rejection_reason: Optional[str] = None

    @property
    def is_pending(self) -> bool:
        """Check if approval is still pending."""
        return self.approved_by is None and self.rejected_by is None

    @property
    def is_expired(self) -> bool:
        """Check if the approval request has expired."""
        if self.expires_at is None:
            return False
        return datetime.now() > self.expires_at
