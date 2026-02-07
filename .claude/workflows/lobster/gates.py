#!/usr/bin/env python3
"""
Lobster Workflow Engine - Approval Gates

Implementations of different approval gate types:
- ManualApprovalGate: Requires explicit user approval
- TimeoutGate: Auto-approves after a duration
- ConditionalGate: Approves based on step outputs
"""

from datetime import datetime, timedelta, timezone
from typing import Any, Optional
import re

from .types import (
    ApprovalGate,
    ApprovalRequest,
    GateType,
    StepResult,
)


def create_approval_request(
    workflow_id: str,
    step_name: str,
    gate: ApprovalGate,
) -> ApprovalRequest:
    """
    Create an approval request for a gate.

    Args:
        workflow_id: ID of the workflow
        step_name: Name of the step with the gate
        gate: The gate configuration

    Returns:
        An ApprovalRequest ready to be saved
    """
    now = datetime.now(timezone.utc)

    expires_at = None
    if gate.gate_type == GateType.TIMEOUT and gate.timeout_seconds:
        expires_at = now + timedelta(seconds=gate.timeout_seconds)

    return ApprovalRequest(
        workflow_id=workflow_id,
        step_name=step_name,
        gate=gate,
        requested_at=now,
        expires_at=expires_at,
    )


def check_timeout_gate(request: ApprovalRequest) -> bool:
    """
    Check if a timeout gate has auto-approved.

    Args:
        request: The approval request

    Returns:
        True if auto-approved, False if still waiting
    """
    if request.gate.gate_type != GateType.TIMEOUT:
        return False

    if request.expires_at is None:
        return False

    return datetime.now(timezone.utc) >= request.expires_at


def evaluate_condition(
    condition: str,
    step_results: list[StepResult],
    variables: dict[str, Any],
) -> bool:
    """
    Evaluate a conditional gate expression.

    The condition can reference:
    - step_results: Dict of step_name -> StepResult
    - variables: Dict of variable names -> values
    - findings: Convenience alias for code review findings

    Example conditions:
    - "findings.critical == 0"
    - "test_coverage > 80"
    - "steps['review'].status == 'completed'"

    Args:
        condition: The condition expression
        step_results: Results from previous steps
        variables: Workflow variables

    Returns:
        True if condition is met, False otherwise
    """
    # Build evaluation context
    results_dict = {r.step_name: r for r in step_results}

    # Extract common patterns from results
    findings = {"critical": 0, "high": 0, "medium": 0, "low": 0}
    for result in step_results:
        if result.output and isinstance(result.output, dict):
            if "findings" in result.output:
                for f in result.output.get("findings", []):
                    severity = f.get("severity", "low")
                    findings[severity] = findings.get(severity, 0) + 1

    # Build safe evaluation context
    context = {
        "steps": results_dict,
        "findings": type("Findings", (), findings)(),  # Allow dot notation
        "variables": variables,
        **variables,  # Also allow direct variable access
    }

    # Simple condition evaluation (avoid eval for security)
    # Support basic comparisons: ==, !=, <, >, <=, >=

    # Pattern: left op right
    # Updated to support hyphens in identifiers and bracket notation
    # Examples: "findings.critical == 0", "steps.test-unit.status == 'completed'",
    #           "steps['test-unit'].outputs.coverage > 80"
    pattern = r"([\w\-]+(?:\.[\w\-]+|\[['\"]\w+[-\w]*['\"]\])*)\s*(==|!=|<|>|<=|>=)\s*(\d+|'[^']*'|\"[^\"]*\")"
    match = re.match(pattern, condition.strip())

    if not match:
        # Unsupported condition format
        return False

    left_expr, op, right_expr = match.groups()

    # Resolve left side
    left_value = _resolve_expression(left_expr, context)
    if left_value is None:
        return False

    # Resolve right side
    if right_expr.startswith(("'", '"')):
        right_value = right_expr[1:-1]  # Remove quotes
    else:
        right_value = int(right_expr)

    # Compare
    if op == "==":
        return left_value == right_value
    elif op == "!=":
        return left_value != right_value
    elif op == "<":
        return left_value < right_value
    elif op == ">":
        return left_value > right_value
    elif op == "<=":
        return left_value <= right_value
    elif op == ">=":
        return left_value >= right_value

    return False


def _resolve_expression(expr: str, context: dict[str, Any]) -> Any:
    """
    Resolve a dot-notation or bracket-notation expression against a context.

    Args:
        expr: Expression like "findings.critical", "steps.test-unit.status",
              or "steps['test-unit'].status"
        context: Dictionary of available values

    Returns:
        The resolved value, or None if not found
    """
    value = context

    # Handle bracket notation: steps['test-unit'].status
    # Split by bracket notation first
    bracket_pattern = r"\['([^']+)'\]|\[\"([^\"]+)\"\]"
    parts = []
    current_expr = expr

    while current_expr:
        # Check for bracket notation
        match = re.search(bracket_pattern, current_expr)
        if match:
            # Get everything before the bracket
            before = current_expr[: match.start()]
            if before and before != ".":
                # Split by dots and filter out empty parts
                parts.extend([p for p in before.split(".") if p])

            # Add the bracketed key
            key = match.group(1) or match.group(2)
            parts.append(key)

            # Continue with remainder
            current_expr = current_expr[match.end() :]
            if current_expr.startswith("."):
                current_expr = current_expr[1:]
        else:
            # No more brackets, split remaining by dots
            if current_expr:
                parts.extend([p for p in current_expr.split(".") if p])
            break

    # Navigate through the parts
    for part in parts:
        if isinstance(value, dict) and part in value:
            value = value[part]
        elif hasattr(value, part):
            value = getattr(value, part)
        else:
            return None

    return value


def approve_request(
    request: ApprovalRequest,
    approved_by: str = "system",
) -> ApprovalRequest:
    """
    Mark a request as approved.

    Args:
        request: The approval request
        approved_by: Who approved it

    Returns:
        Updated approval request
    """
    request.approved_by = approved_by
    request.approved_at = datetime.now(timezone.utc)
    return request


def reject_request(
    request: ApprovalRequest,
    rejected_by: str,
    reason: Optional[str] = None,
) -> ApprovalRequest:
    """
    Mark a request as rejected.

    Args:
        request: The approval request
        rejected_by: Who rejected it
        reason: Optional rejection reason

    Returns:
        Updated approval request
    """
    request.rejected_by = rejected_by
    request.rejected_at = datetime.now(timezone.utc)
    request.rejection_reason = reason
    return request


def format_approval_message(request: ApprovalRequest) -> str:
    """
    Format an approval request as a human-readable message.

    Args:
        request: The approval request

    Returns:
        Formatted message string
    """
    lines = [
        f"## Approval Required: {request.step_name}",
        "",
        f"**Workflow:** {request.workflow_id}",
        f"**Message:** {request.gate.message}",
        "",
    ]

    if request.gate.gate_type == GateType.TIMEOUT and request.expires_at:
        remaining = request.expires_at - datetime.now(timezone.utc)
        minutes = max(0, int(remaining.total_seconds() / 60))
        lines.append(f"*Auto-approves in {minutes} minutes*")
        lines.append("")

    if request.gate.approvers:
        lines.append(f"**Approvers:** {', '.join(request.gate.approvers)}")
        lines.append("")

    lines.extend(
        [
            "### Actions",
            "",
            "- Reply with `/approve` to approve",
            "- Reply with `/reject [reason]` to reject",
        ]
    )

    return "\n".join(lines)
