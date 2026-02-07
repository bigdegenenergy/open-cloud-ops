#!/usr/bin/env python3
"""
Lobster Workflow Engine - Core Engine

The main workflow execution engine that:
- Parses YAML workflow definitions
- Executes steps sequentially with state tracking
- Handles approval gates
- Manages retries and failure handling
"""

from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Optional
import subprocess
import uuid
import yaml
import shlex

from .types import (
    WorkflowDefinition,
    WorkflowStep,
    WorkflowState,
    WorkflowStatus,
    StepResult,
    StepStatus,
    StepType,
    ApprovalGate,
    GateType,
)
from .state import StateManager
from .gates import (
    create_approval_request,
    check_timeout_gate,
    evaluate_condition,
    approve_request,
    format_approval_message,
)


class WorkflowEngine:
    """
    The main workflow execution engine.

    Usage:
        engine = WorkflowEngine()
        workflow = engine.load_workflow("feature-pipeline")
        result = engine.run(workflow, {"feature_description": "Add user auth"})
    """

    def __init__(self, definitions_dir: Optional[Path] = None):
        """
        Initialize the workflow engine.

        Args:
            definitions_dir: Directory containing workflow YAML files.
                           Defaults to .claude/workflows/definitions/
        """
        if definitions_dir is None:
            current_dir = Path(__file__).parent.parent
            definitions_dir = current_dir / "definitions"

        self.definitions_dir = Path(definitions_dir)
        self.state_manager = StateManager()

    def load_workflow(self, name: str) -> WorkflowDefinition:
        """
        Load a workflow definition from YAML.

        Args:
            name: Name of the workflow (without .yaml extension)

        Returns:
            The parsed WorkflowDefinition

        Raises:
            FileNotFoundError: If workflow file doesn't exist
            ValueError: If workflow is invalid
        """
        workflow_file = self.definitions_dir / f"{name}.yaml"

        if not workflow_file.exists():
            raise FileNotFoundError(f"Workflow not found: {workflow_file}")

        with open(workflow_file) as f:
            data = yaml.safe_load(f)

        return self._parse_workflow(data)

    def _parse_workflow(self, data: dict[str, Any]) -> WorkflowDefinition:
        """Parse a workflow definition from YAML data."""
        steps = []
        for step_data in data.get("steps", []):
            # Parse gate if present
            gate = None
            if "gate" in step_data:
                gate_data = step_data["gate"]
                gate = ApprovalGate(
                    gate_type=GateType(gate_data.get("type", "manual")),
                    message=gate_data.get("message", "Approval required"),
                    timeout_seconds=gate_data.get("timeout_seconds"),
                    condition=gate_data.get("condition"),
                    fallback=gate_data.get("fallback", "fail"),
                    approvers=gate_data.get("approvers", []),
                    notify=gate_data.get("notify", True),
                )

            # Determine step type
            step_type = StepType.COMMAND
            target = ""

            if "command" in step_data:
                step_type = StepType.COMMAND
                target = step_data["command"]
            elif "agent" in step_data:
                step_type = StepType.AGENT
                target = step_data["agent"]
            elif "shell" in step_data:
                step_type = StepType.SHELL
                target = step_data["shell"]
            elif "parallel" in step_data:
                step_type = StepType.PARALLEL
                target = ",".join(step_data["parallel"])

            step = WorkflowStep(
                name=step_data["name"],
                step_type=step_type,
                target=target,
                inputs=step_data.get("inputs", {}),
                outputs=step_data.get("outputs", []),
                gate=gate,
                timeout_seconds=step_data.get("timeout_seconds", 600),
                retry_count=step_data.get("retry_count", 0),
                continue_on_failure=step_data.get("continue_on_failure", False),
                depends_on=step_data.get("depends_on", []),
            )
            steps.append(step)

        return WorkflowDefinition(
            name=data["name"],
            description=data.get("description", ""),
            version=data.get("version", "1.0.0"),
            steps=steps,
            on_failure=data.get("on_failure", "notify"),
            on_success=data.get("on_success", "notify"),
            timeout_seconds=data.get("timeout_seconds", 3600),
            metadata=data.get("metadata", {}),
        )

    def start(
        self,
        workflow: WorkflowDefinition,
        variables: Optional[dict[str, Any]] = None,
    ) -> WorkflowState:
        """
        Start a new workflow execution.

        Args:
            workflow: The workflow to execute
            variables: Initial variables for template substitution

        Returns:
            The initial workflow state
        """
        workflow_id = f"{workflow.name}-{uuid.uuid4().hex[:8]}"

        state = WorkflowState(
            workflow_name=workflow.name,
            workflow_id=workflow_id,
            status=WorkflowStatus.NOT_STARTED,
            variables=variables or {},
            started_at=datetime.now(timezone.utc),
        )

        self.state_manager.save_state(state)
        return state

    def execute_step(
        self,
        workflow: WorkflowDefinition,
        state: WorkflowState,
        step_index: int,
    ) -> tuple[WorkflowState, bool]:
        """
        Execute a single workflow step.

        Args:
            workflow: The workflow definition
            state: Current workflow state
            step_index: Index of the step to execute

        Returns:
            Tuple of (updated state, should_continue)
        """
        if step_index >= len(workflow.steps):
            state.status = WorkflowStatus.COMPLETED
            state.completed_at = datetime.now(timezone.utc)
            self.state_manager.save_state(state)
            return state, False

        step = workflow.steps[step_index]
        state.current_step = step_index
        state.status = WorkflowStatus.RUNNING

        # Create step result
        result = StepResult(
            step_name=step.name,
            status=StepStatus.RUNNING,
            started_at=datetime.now(timezone.utc),
        )

        # Execute based on step type
        try:
            output = self._execute_step_target(step, state.variables)
            result.status = StepStatus.COMPLETED
            result.output = output
        except Exception as e:
            result.status = StepStatus.FAILED
            result.error = str(e)

            if not step.continue_on_failure:
                state.status = WorkflowStatus.FAILED
                state.error = f"Step '{step.name}' failed: {e}"
                result.completed_at = datetime.now(timezone.utc)
                state.step_results.append(result)
                self.state_manager.save_state(state)
                return state, False

        result.completed_at = datetime.now(timezone.utc)
        result.duration_seconds = (
            result.completed_at - result.started_at
        ).total_seconds()
        state.step_results.append(result)

        # Check for approval gate
        if step.gate and result.status == StepStatus.COMPLETED:
            should_pause = self._handle_gate(workflow, state, step, result)
            if should_pause:
                return state, False

        self.state_manager.save_state(state)
        return state, True

    def _execute_step_target(
        self,
        step: WorkflowStep,
        variables: dict[str, Any],
    ) -> Any:
        """
        Execute the target of a step.

        Execution behavior by step type:
        - SHELL: Execute shell command via subprocess
        - COMMAND: Return command info (requires Claude integration)
        - AGENT: Return agent info (requires Claude integration)
        - PARALLEL: Execute multiple targets concurrently

        Args:
            step: The step to execute
            variables: Variables for template substitution

        Returns:
            The output of the step

        Raises:
            RuntimeError: If step execution fails
        """
        # For shell commands, use shell-safe variable substitution
        # to prevent command injection attacks
        shell_safe = step.step_type == StepType.SHELL

        # Template substitution in inputs
        inputs = self._substitute_variables(
            step.inputs, variables, shell_safe=shell_safe
        )

        # For shell commands, also substitute and escape the target command
        target = step.target
        if shell_safe:
            target = self._substitute_variables(target, variables, shell_safe=True)

        # Execute based on step type
        if step.step_type == StepType.SHELL:
            return self._execute_shell(target, step.timeout_seconds)
        elif step.step_type == StepType.COMMAND:
            # COMMAND type requires Claude integration
            # Return info for external orchestration
            return {
                "step_type": step.step_type.value,
                "command": target,
                "inputs": inputs,
                "requires": "claude_integration",
            }
        elif step.step_type == StepType.AGENT:
            # AGENT type requires Claude subagent invocation
            # Return info for external orchestration
            return {
                "step_type": step.step_type.value,
                "agent": target,
                "inputs": inputs,
                "requires": "claude_integration",
            }
        elif step.step_type == StepType.PARALLEL:
            # PARALLEL type - targets are comma-separated
            targets = target.split(",")
            return {
                "step_type": step.step_type.value,
                "targets": targets,
                "inputs": inputs,
                "requires": "parallel_executor",
            }
        else:
            raise ValueError(f"Unknown step type: {step.step_type}")

    def _execute_shell(
        self,
        command: str,
        timeout_seconds: int = 600,
    ) -> dict[str, Any]:
        """
        Execute a shell command via subprocess.

        Args:
            command: The shell command to execute
            timeout_seconds: Maximum execution time

        Returns:
            Dict with stdout, stderr, and return_code

        Raises:
            RuntimeError: If command fails with non-zero exit code
        """
        try:
            result = subprocess.run(
                command,
                shell=True,
                capture_output=True,
                text=True,
                timeout=timeout_seconds,
            )

            output = {
                "stdout": result.stdout,
                "stderr": result.stderr,
                "return_code": result.returncode,
            }

            if result.returncode != 0:
                raise RuntimeError(
                    f"Shell command failed with exit code {result.returncode}: "
                    f"{result.stderr or result.stdout}"
                )

            return output

        except subprocess.TimeoutExpired as e:
            raise RuntimeError(f"Shell command timed out after {timeout_seconds}s: {e}")
        except Exception as e:
            raise RuntimeError(f"Shell command execution failed: {e}")

    def _substitute_variables(
        self,
        value: Any,
        variables: dict[str, Any],
        shell_safe: bool = False,
    ) -> Any:
        """
        Substitute {{ variable }} patterns in values.

        Args:
            value: The value to substitute variables in
            variables: Dictionary of variable names to values
            shell_safe: If True, escape values for safe shell execution

        Returns:
            Value with variables substituted
        """
        if isinstance(value, str):
            for var_name, var_value in variables.items():
                # Convert variable value to string
                var_str = str(var_value)

                # For shell commands, escape the value to prevent injection
                if shell_safe:
                    var_str = shlex.quote(var_str)

                value = value.replace(f"{{{{ {var_name} }}}}", var_str)
            return value
        elif isinstance(value, dict):
            return {
                k: self._substitute_variables(v, variables, shell_safe)
                for k, v in value.items()
            }
        elif isinstance(value, list):
            return [self._substitute_variables(v, variables, shell_safe) for v in value]
        return value

    def _handle_gate(
        self,
        workflow: WorkflowDefinition,
        state: WorkflowState,
        step: WorkflowStep,
        result: StepResult,
    ) -> bool:
        """
        Handle an approval gate after a step.

        Args:
            workflow: The workflow definition
            state: Current workflow state
            step: The step with the gate
            result: The step result

        Returns:
            True if workflow should pause for approval
        """
        gate = step.gate
        if gate is None:
            return False

        # Check conditional gate
        if gate.gate_type == GateType.CONDITIONAL:
            if gate.condition:
                passed = evaluate_condition(
                    gate.condition,
                    state.step_results,
                    state.variables,
                )
                if passed:
                    return False  # Condition met, continue
                elif gate.fallback == "fail":
                    state.status = WorkflowStatus.FAILED
                    state.error = f"Gate condition not met: {gate.condition}"
                    self.state_manager.save_state(state)
                    return True
                elif gate.fallback == "skip":
                    return False  # Skip remaining steps
                # fallback == "continue" - wait for approval

        # Create approval request
        request = create_approval_request(
            workflow_id=state.workflow_id,
            step_name=step.name,
            gate=gate,
        )

        # Save request and pause workflow
        self.state_manager.save_approval_request(request)
        state.status = WorkflowStatus.PAUSED
        state.pending_approval = step.name
        self.state_manager.save_state(state)

        # Print approval message
        print(format_approval_message(request))

        return True

    def resume(self, workflow_id: str) -> Optional[WorkflowState]:
        """
        Resume a paused workflow after approval.

        Args:
            workflow_id: ID of the workflow to resume

        Returns:
            The updated workflow state, or None if not found
        """
        state = self.state_manager.load_state(workflow_id)
        if state is None:
            return None

        if state.status != WorkflowStatus.PAUSED:
            return state

        # Check for timeout-based auto-approval
        request = self.state_manager.load_approval_request(workflow_id)
        if request and check_timeout_gate(request):
            approve_request(request, approved_by="timeout")
            self.state_manager.save_approval_request(request)

        # Check if approved
        if request and not request.is_pending:
            state.status = WorkflowStatus.RUNNING
            state.pending_approval = None
            self.state_manager.save_state(state)

        return state

    def approve(
        self,
        workflow_id: str,
        approved_by: str = "user",
    ) -> Optional[WorkflowState]:
        """
        Approve a pending workflow gate.

        Args:
            workflow_id: ID of the workflow
            approved_by: Who approved it

        Returns:
            The updated workflow state, or None if not found
        """
        request = self.state_manager.load_approval_request(workflow_id)
        if request is None or not request.is_pending:
            return None

        approve_request(request, approved_by=approved_by)
        self.state_manager.save_approval_request(request)

        state = self.state_manager.load_state(workflow_id)
        if state:
            state.status = WorkflowStatus.RUNNING
            state.pending_approval = None
            self.state_manager.save_state(state)

        return state

    def reject(
        self,
        workflow_id: str,
        rejected_by: str = "user",
        reason: Optional[str] = None,
    ) -> Optional[WorkflowState]:
        """
        Reject a pending workflow gate.

        Args:
            workflow_id: ID of the workflow
            rejected_by: Who rejected it
            reason: Optional rejection reason

        Returns:
            The updated workflow state, or None if not found
        """
        from .gates import reject_request

        request = self.state_manager.load_approval_request(workflow_id)
        if request is None or not request.is_pending:
            return None

        reject_request(request, rejected_by=rejected_by, reason=reason)
        self.state_manager.save_approval_request(request)

        state = self.state_manager.load_state(workflow_id)
        if state:
            state.status = WorkflowStatus.FAILED
            state.error = f"Rejected by {rejected_by}: {reason or 'No reason provided'}"
            self.state_manager.save_state(state)

        return state

    def status(self, workflow_id: str) -> Optional[WorkflowState]:
        """Get the current status of a workflow."""
        return self.state_manager.load_state(workflow_id)

    def list_active(self) -> list[WorkflowState]:
        """List all active workflows."""
        return self.state_manager.list_active_workflows()

    def list_pending_approvals(self):
        """List all pending approval requests."""
        return self.state_manager.list_pending_approvals()
