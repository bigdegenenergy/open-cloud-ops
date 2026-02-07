# Workflow: Execute Typed Pipeline

Execute a predefined workflow with approval gates.

## Target

$ARGUMENTS

Format: `workflow-name [--var key=value]`

Examples:

- `feature-pipeline --var description="Add user auth"`
- `security-audit --var target=src/auth/`
- `deploy-staging`

## Available Workflows

```bash
# List available workflows
ls -1 .claude/workflows/definitions/*.yaml 2>/dev/null | xargs -I{} basename {} .yaml || echo "No workflows defined"
```

## Workflow Execution

### Step 1: Load Workflow

```python
from .claude.workflows.lobster import WorkflowEngine

engine = WorkflowEngine()
workflow = engine.load_workflow("$WORKFLOW_NAME")
```

### Step 2: Start Execution

```python
state = engine.start(workflow, {
    # Variables from --var arguments
})
```

### Step 3: Execute Steps

For each step in the workflow:

1. Execute the step (command, agent, or shell)
2. Check for approval gates
3. If gate requires approval, pause and wait
4. Continue to next step

### Step 4: Handle Approvals

When a gate is reached:

```
## Approval Required: review

**Workflow:** feature-pipeline-abc123
**Message:** Tests passing. Review implementation before proceeding?

### Actions

- Reply with `/approve` to approve
- Reply with `/reject [reason]` to reject
```

## Workflow Definition Format

```yaml
# .claude/workflows/definitions/feature-pipeline.yaml
name: feature-pipeline
description: Full feature implementation with quality gates
version: "1.0.0"

steps:
  - name: plan
    command: /plan
    inputs:
      description: "{{ description }}"

  - name: implement
    agent: "@typescript-pro"
    inputs:
      task: "Implement the plan"
    timeout_seconds: 1800

  - name: test
    command: /qa
    gate:
      type: manual
      message: "Tests passing. Review implementation?"

  - name: review
    agent: "@code-reviewer"
    gate:
      type: conditional
      condition: "findings.critical == 0"
      fallback: fail

  - name: ship
    command: /ship
    gate:
      type: manual
      message: "Ready to commit and push?"

on_failure: notify
on_success: notify
timeout_seconds: 7200
```

## Gate Types

| Type        | Behavior                      | Config                  |
| ----------- | ----------------------------- | ----------------------- |
| manual      | Waits for explicit approval   | `message`, `approvers`  |
| timeout     | Auto-approves after duration  | `timeout_seconds`       |
| conditional | Approves if condition is true | `condition`, `fallback` |

## Workflow Status

Check current workflow status with `/workflow-status`.

## Execution Report

After workflow completes:

```markdown
## Workflow Complete: feature-pipeline

**ID:** feature-pipeline-abc123
**Duration:** 15 minutes
**Status:** COMPLETED

### Steps

| Step      | Status    | Duration |
| --------- | --------- | -------- |
| plan      | completed | 2m       |
| implement | completed | 8m       |
| test      | completed | 3m       |
| review    | completed | 1m       |
| ship      | completed | 1m       |

### Output

[Summary of workflow results]
```

---

**Execute workflow:** $ARGUMENTS
