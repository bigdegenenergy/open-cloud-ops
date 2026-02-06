# Workflow Approve: Continue Paused Pipeline

Approve a pending workflow gate to continue execution.

## Target

$ARGUMENTS

Format: `workflow-id [--user username]`

Examples:

- `feature-pipeline-abc123`
- `security-audit-def456 --user admin`

## Approval Process

### Step 1: Verify Pending Approval

```bash
# Check if workflow has pending approval
if [ -f ".claude/artifacts/workflow-state/$WORKFLOW_ID.approval.json" ]; then
  echo "Found pending approval"
  cat ".claude/artifacts/workflow-state/$WORKFLOW_ID.approval.json" | head -20
else
  echo "No pending approval for $WORKFLOW_ID"
fi
```

### Step 2: Approve Gate

```bash
# Export variables to environment to prevent injection
export WORKFLOW_ID="$WORKFLOW_ID"
export USER="$USER"

# Execute Python safely using environment variables
python3 -c '
import os, sys, importlib.util

workflow_id = os.environ.get("WORKFLOW_ID", "")
user = os.environ.get("USER", "")

# Load from hidden directory using importlib (dot-prefixed dirs are not valid packages)
spec = importlib.util.spec_from_file_location(
    "workflow_engine",
    os.path.join(os.getcwd(), ".claude", "workflows", "lobster.py")
)
mod = importlib.util.module_from_spec(spec)
spec.loader.exec_module(mod)

engine = mod.WorkflowEngine()
state = engine.approve(workflow_id, approved_by=user)
'
```

### Step 3: Resume Execution

After approval, the workflow resumes from the next step.

## Alternative: Reject

To reject instead of approving:

```
/workflow-reject $WORKFLOW_ID "Reason for rejection"
```

## Approval Confirmation

```markdown
## Approval Confirmed

**Workflow:** feature-pipeline-abc123
**Step:** test
**Approved by:** user
**Approved at:** 2026-01-25 10:35:00

Workflow resuming...

### Next Step

**Step:** review
**Agent:** @code-reviewer
```

## Rejection Confirmation

```markdown
## Workflow Rejected

**Workflow:** feature-pipeline-abc123
**Step:** test
**Rejected by:** user
**Reason:** Need to add more test coverage

Workflow status: FAILED
```

## Notes

- Only pending workflows can be approved
- Timeout gates auto-approve after their duration
- Conditional gates may auto-approve based on conditions
- Multiple approvers can be configured per gate

---

**Approve workflow:** $ARGUMENTS
