# Ralph Coder Skill

> Enhanced autonomous coding loop with quality gates, test-first enforcement, and atomic commits.

**Extends:** `autonomous-loop` skill (read that first for core concepts)

## What Ralph Coder Adds

| Feature              | Base autonomous-loop | Ralph Coder               |
| -------------------- | -------------------- | ------------------------- |
| Exit gates           | Dual-condition       | Same                      |
| Circuit breaker      | Yes                  | Yes                       |
| **Quality gates**    | No                   | Lint/format between loops |
| **Test-first**       | Optional             | Enforced                  |
| **Atomic commits**   | No                   | Commit after each task    |
| **Progress metrics** | Basic                | Detailed tracking         |

## The Ralph Coder Protocol

### Loop Structure

```
┌─────────────────────────────────────┐
│ 1. PLAN (what to do this loop)      │
├─────────────────────────────────────┤
│ 2. TEST (write failing test first)  │
├─────────────────────────────────────┤
│ 3. CODE (make the test pass)        │
├─────────────────────────────────────┤
│ 4. QUALITY (lint, format, type-check) │
├─────────────────────────────────────┤
│ 5. VERIFY (all tests green)         │
├─────────────────────────────────────┤
│ 6. COMMIT (atomic commit if green)  │
├─────────────────────────────────────┤
│ 7. REPORT (status + metrics)        │
└─────────────────────────────────────┘
         ↓
    Next loop or EXIT
```

### Step 1: PLAN

Before coding, state:

- What specific task this loop addresses
- Expected outcome
- How you'll verify success

```markdown
### Loop N Plan

**Task:** Implement user authentication
**Outcome:** Users can log in with email/password
**Verification:** Auth tests pass, manual login works
```

### Step 2: TEST (Red Phase)

**MANDATORY**: Write a failing test before implementation.

```bash
# Write the test
# Run to confirm it fails
npm test -- --grep "user auth"
# Expected: FAIL (test exists, code doesn't)
```

If test already exists and passing, you're in maintenance mode (skip to Step 3).

### Step 3: CODE (Green Phase)

Implement just enough code to make the test pass.

**Rules:**

- Minimal implementation
- No premature optimization
- No unrelated changes
- One feature per loop

### Step 4: QUALITY (Quality Gate)

**MANDATORY**: Run quality checks before proceeding.

```bash
# TypeScript/JavaScript
npm run lint
npm run format:check
npm run typecheck

# Python
ruff check .
black --check .
mypy .

# Go
go vet ./...
gofmt -d .

# Rust
cargo clippy
cargo fmt --check
```

**If quality fails:**

1. Fix the issues
2. Re-run quality checks
3. Do NOT proceed until green

### Step 5: VERIFY

Run full test suite:

```bash
npm test
# or pytest, cargo test, go test, etc.
```

**All tests must pass before proceeding.**

### Step 6: COMMIT (Atomic Commit)

If Steps 4-5 pass, create atomic commit:

```bash
git add -A
git commit -m "feat: [description of this loop's work]"
```

**Commit message format:**

- `feat:` for new features
- `fix:` for bug fixes
- `refactor:` for code improvements
- `test:` for test-only changes

**Benefits of atomic commits:**

- Easy to revert single changes
- Clear history of progress
- Bisect-friendly

### Step 7: REPORT (Enhanced Status)

```markdown
## Loop N Status Report

### Metrics

| Metric        | Before | After | Delta |
| ------------- | ------ | ----- | ----- |
| Lines of code | 1247   | 1289  | +42   |
| Test count    | 23     | 25    | +2    |
| Test coverage | 78%    | 81%   | +3%   |
| Lint errors   | 0      | 0     | 0     |

### This Loop

- **Task:** [what was planned]
- **Result:** [DONE / PARTIAL / BLOCKED]
- **Tests Added:** [count]
- **Files Modified:** [list]
- **Commit:** [hash] "[message]"

### Quality Gates

- [x] Lint passed
- [x] Format passed
- [x] Types passed
- [x] Tests passed

### Progress

STATUS: IN_PROGRESS
LOOP: N
EXIT_SIGNAL: false
NEXT: [next task]
```

## Test-First Enforcement

### Why Test-First?

| Without Test-First   | With Test-First         |
| -------------------- | ----------------------- |
| "I think it works"   | "Test proves it works"  |
| Regression risk      | Regression-protected    |
| Hard to refactor     | Safe to refactor        |
| Unclear requirements | Executable requirements |

### Test-First Patterns

**For new features:**

```python
# 1. Write test
def test_user_can_login():
    user = create_user("test@example.com", "password")
    result = login("test@example.com", "password")
    assert result.success
    assert result.user == user

# 2. Run test (should FAIL)
# 3. Implement login()
# 4. Run test (should PASS)
```

**For bug fixes:**

```python
# 1. Write test that reproduces the bug
def test_handles_empty_input():
    # This was crashing before
    result = process_input("")
    assert result is None  # Should not crash

# 2. Run test (should FAIL - reproduces bug)
# 3. Fix the bug
# 4. Run test (should PASS - bug fixed)
```

### When Test-First is Hard

If you can't write a test first:

1. Document why (e.g., "UI change, no test framework")
2. Add manual verification steps
3. Add regression test after implementation if possible

## Progress Metrics

Track across loops:

```markdown
## Session Metrics

| Metric      | Start | Current | Target | Progress |
| ----------- | ----- | ------- | ------ | -------- |
| Tasks       | 0/5   | 3/5     | 5/5    | 60%      |
| Tests       | 12    | 18      | 20+    | 90%      |
| Coverage    | 72%   | 81%     | 80%    | ✅       |
| Lint errors | 3     | 0       | 0      | ✅       |
| Commits     | 0     | 3       | -      | -        |

## Velocity

- Loops: 4
- Tasks completed: 3
- Avg time per task: ~1 loop
- Blockers hit: 0
```

## Quality Gate Failures

### Lint Failure Protocol

```markdown
### Quality Gate: LINT FAILED

**Errors:**

- src/auth.ts:42 - Unused variable 'temp'
- src/auth.ts:67 - Missing return type

**Action:** Fixing lint errors before proceeding

**Status:** QUALITY_GATE_BLOCKED
```

Fix all errors, then continue loop.

### Test Failure Protocol

```markdown
### Quality Gate: TESTS FAILED

**Failed Tests:**

- test_user_login: AssertionError at line 23
- test_password_hash: Timeout after 5s

**Root Cause Analysis:**

1. test_user_login: Password comparison using == instead of secure compare
2. test_password_hash: Missing await on async function

**Action:** Fixing test failures

**Status:** VERIFY_BLOCKED
```

### Type Check Failure Protocol

```markdown
### Quality Gate: TYPE CHECK FAILED

**Errors:**

- src/user.ts:15 - Type 'string | undefined' not assignable to 'string'

**Action:** Adding proper null handling

**Status:** QUALITY_GATE_BLOCKED
```

## Circuit Breaker Extensions

Beyond base thresholds, Ralph Coder also halts on:

| Trigger               | Threshold     | Action                       |
| --------------------- | ------------- | ---------------------------- |
| Quality gate failures | 3 consecutive | HALT - fix quality issues    |
| Same test failing     | 3 loops       | HALT - step back and rethink |
| No commits            | 5 loops       | HALT - not making progress   |

## Activation Triggers

This skill auto-activates when prompts contain:

- "coder loop", "ralph mode"
- "autonomous coding", "code until done"
- "TDD loop", "test-driven development"
- "implement and commit"

## Integration

Works with:

- **autonomous-loop**: Base patterns
- **tdd**: Test-first methodology
- **refactoring**: Safe changes after green
- **deslop**: Simplify after each commit

## Example Session

```
Loop 1:
  Task: Add user registration endpoint
  Test: test_user_registration (FAILING)
  Code: Implement /api/register
  Quality: ✅ lint, ✅ format, ✅ types
  Verify: ✅ all tests pass
  Commit: abc123 "feat: add user registration endpoint"
  NEXT: Add email verification

Loop 2:
  Task: Add email verification
  Test: test_email_verification (FAILING)
  Code: Implement verification flow
  Quality: ✅ lint, ✅ format, ✅ types
  Verify: ✅ all tests pass
  Commit: def456 "feat: add email verification"
  NEXT: Add password reset

Loop 3:
  Task: Add password reset
  Test: test_password_reset (FAILING)
  Code: Implement password reset
  Quality: ✅ lint, ✅ format, ✅ types
  Verify: ✅ all tests pass
  Commit: ghi789 "feat: add password reset"

  EXIT_SIGNAL: true (all tasks complete)
```
