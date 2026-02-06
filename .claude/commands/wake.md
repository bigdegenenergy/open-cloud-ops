---
description: Wake up to review feedback. Pull, read instructions, fix code, delete file, commit.
model: haiku
allowed-tools: Bash(*), Read(*), Edit(*), Write(*), Grep(*), Glob(*)
---

# Wake Protocol: Review Feedback Handler

You are waking up to address review feedback from the Review Agent (Gemini).

## Step 1: Pull Latest Changes

```bash
git pull origin $(git branch --show-current)
```

## Step 2: Check for Review Instructions

Look for `REVIEW_INSTRUCTIONS.md` in the repository root.

!`ls -la REVIEW_INSTRUCTIONS.md 2>/dev/null || echo "No REVIEW_INSTRUCTIONS.md found"`

## Step 3: Process Instructions (if file exists)

If `REVIEW_INSTRUCTIONS.md` exists:

### 3a. Read the Instructions

Read the file carefully. It contains:

- TOML-formatted issues from the Review Agent
- Severity levels (critical, important, suggestion)
- File locations and descriptions

### 3b. Fix the Code

For each issue:

1. Navigate to the specified file and location
2. Understand the concern
3. Implement the fix
4. Verify the fix addresses the issue

**Priority order:** critical > important > suggestion

### 3c. Delete the Instructions File

After addressing all issues:

```bash
git rm REVIEW_INSTRUCTIONS.md
```

### 3d. Commit with Agent-Note Trailer

Commit your changes with this format:

```
<type>: <subject>

<optional body explaining changes>

Agent-Note: <summary of fixes for the Review Agent>
```

Example:

```
fix: address review feedback

Updated input validation and added null checks.

Agent-Note: Fixed SQL injection by using parameterized queries. Added null check for user input in auth.ts:42.
```

## Step 4: Push Changes

```bash
git push
```

## If No Instructions Found

If `REVIEW_INSTRUCTIONS.md` does not exist:

1. Check if there are any PR review comments to address
2. Report that no pending review instructions were found
3. Ask if there's specific feedback to address

## Important Rules

- **Be thorough** - Address ALL issues, not just critical ones
- **Be explicit** - Your Agent-Note should reference specific issues fixed
- **Be clean** - Always delete REVIEW_INSTRUCTIONS.md before committing
- **Be responsive** - The Review Agent will verify your fixes on re-review

## Exit Conditions

Return control when:

1. All issues addressed, file deleted, changes committed and pushed
2. No REVIEW_INSTRUCTIONS.md found (report status)
3. Blocked by an issue requiring human decision

**The Review Agent is watching. Make your fixes count.**
