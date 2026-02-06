# Workflow Status: Check Pipeline Progress

Check the status of running and paused workflows.

## Target

$ARGUMENTS

Format: `[workflow-id]` or empty for all active

Examples:

- (empty) - Show all active workflows
- `feature-pipeline-abc123` - Show specific workflow

## Active Workflows

```bash
# List workflow state files
if [ -d .claude/artifacts/workflow-state ]; then
  echo "### Active Workflows"
  for f in .claude/artifacts/workflow-state/*.json; do
    if [ -f "$f" ] && [[ ! "$f" == *".approval.json" ]]; then
      echo "- $(basename "$f" .json)"
    fi
  done 2>/dev/null || echo "No active workflows"
else
  echo "No workflow state directory found"
fi
```

## Pending Approvals

```bash
# List pending approvals
if [ -d .claude/artifacts/workflow-state ]; then
  echo "### Pending Approvals"
  for f in .claude/artifacts/workflow-state/*.approval.json; do
    if [ -f "$f" ]; then
      echo "- $(basename "$f" .approval.json)"
    fi
  done 2>/dev/null || echo "No pending approvals"
fi
```

## Status Output Format

```markdown
## Workflow Status: feature-pipeline-abc123

**Name:** feature-pipeline
**Status:** PAUSED (waiting for approval)
**Started:** 2026-01-25 10:30:00
**Duration:** 5 minutes

### Steps Progress

| #   | Step      | Status           | Duration |
| --- | --------- | ---------------- | -------- |
| 1   | plan      | completed        | 2m       |
| 2   | implement | completed        | 3m       |
| 3   | test      | waiting_approval | -        |
| 4   | review    | pending          | -        |
| 5   | ship      | pending          | -        |

### Pending Approval

**Step:** test
**Message:** Tests passing. Review implementation before proceeding?

**Actions:**

- `/workflow-approve feature-pipeline-abc123` to approve
- `/workflow-reject feature-pipeline-abc123 [reason]` to reject
```

## Status Codes

| Status      | Meaning                            |
| ----------- | ---------------------------------- |
| NOT_STARTED | Workflow created but not started   |
| RUNNING     | Currently executing steps          |
| PAUSED      | Waiting for approval at a gate     |
| COMPLETED   | All steps finished successfully    |
| FAILED      | A step failed and workflow stopped |
| CANCELLED   | Manually cancelled by user         |

## Quick Actions

```bash
# Approve pending workflow
/workflow-approve <workflow-id>

# Reject pending workflow
/workflow-reject <workflow-id> "Reason for rejection"

# Cancel running workflow
/workflow-cancel <workflow-id>
```

---

**Check status of:** $ARGUMENTS
