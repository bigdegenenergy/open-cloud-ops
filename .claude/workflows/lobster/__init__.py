"""
Lobster Workflow Engine

A typed workflow runtime with approval gates for deterministic,
multi-step pipelines that can pause for human approval.

Usage:
    from .claude.workflows.lobster import WorkflowEngine

    engine = WorkflowEngine()
    workflow = engine.load_workflow("feature-pipeline")
    state = engine.start(workflow, {"description": "Add user auth"})

    # Execute steps
    while state.status == WorkflowStatus.RUNNING:
        state, should_continue = engine.execute_step(workflow, state, state.current_step)
        if not should_continue:
            break

    # If paused for approval
    if state.status == WorkflowStatus.PAUSED:
        engine.approve(state.workflow_id)
        state = engine.resume(state.workflow_id)
"""

from .types import (
    WorkflowDefinition,
    WorkflowStep,
    WorkflowState,
    WorkflowStatus,
    StepResult,
    StepStatus,
    StepType,
    ApprovalGate,
    ApprovalRequest,
    GateType,
)
from .engine import WorkflowEngine
from .state import StateManager

__all__ = [
    "WorkflowEngine",
    "StateManager",
    "WorkflowDefinition",
    "WorkflowStep",
    "WorkflowState",
    "WorkflowStatus",
    "StepResult",
    "StepStatus",
    "StepType",
    "ApprovalGate",
    "ApprovalRequest",
    "GateType",
]
