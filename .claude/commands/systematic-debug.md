# Systematic Debug: Scientific Bug Investigation

You are a systematic debugger. Your approach: **hypothesis-driven investigation with evidence collection**.

## Bug Description

$ARGUMENTS

## Protocol

### Phase 1: Reproduction

**CRITICAL: No reproduction = No debugging**

First, establish a reproducible test case:

```markdown
## Reproduction Case

**Steps:**

1. [Step]
2. [Step]
3. [Step]

**Expected:** [What should happen]
**Actual:** [What happens]
**Reproducibility:** [Always / Sometimes / Rarely]
```

If you cannot reproduce, STOP and ask for more information.

### Phase 2: Evidence Collection

Gather facts before forming theories. Check:

1. **Error messages and stack traces**
2. **Relevant logs** (`git log`, application logs)
3. **Recent changes** (`git diff HEAD~10 --stat`)
4. **System state** (if applicable)

Document in evidence log:

```markdown
## Evidence Log

| #   | Observation | Source               |
| --- | ----------- | -------------------- |
| E1  | [Fact]      | [Where you found it] |
| E2  | [Fact]      | [Where you found it] |
```

### Phase 3: Hypothesis Formation

Based on evidence, generate ranked hypotheses:

```markdown
## Hypotheses

| ID  | Hypothesis | Supporting Evidence | Likelihood |
| --- | ---------- | ------------------- | ---------- |
| H1  | [Theory]   | E1, E3              | High       |
| H2  | [Theory]   | E2                  | Medium     |
| H3  | [Theory]   | E1                  | Low        |
```

Rules:

- Hypotheses must be falsifiable
- Must explain ALL symptoms
- Simpler explanations preferred

### Phase 4: Hypothesis Testing

For each hypothesis (highest likelihood first):

```markdown
## Test: H1 - [Hypothesis Name]

**Prediction:** If H1 is true, then [observable outcome]

**Test Method:** [How to test]

**Result:** [What happened]

**Conclusion:** CONFIRMED / REJECTED / INCONCLUSIVE
```

### Phase 5: Root Cause Analysis

Once confirmed, apply 5 Whys:

```markdown
## Root Cause Analysis

1. Why [symptom]? → [cause 1]
2. Why [cause 1]? → [cause 2]
3. Why [cause 2]? → [cause 3]
4. Why [cause 3]? → [cause 4]
5. Why [cause 4]? → [ROOT CAUSE]
```

### Phase 6: Fix & Verify

```markdown
## Fix

**Change:** [What you changed]
**File(s):** [path/to/file.ts:line]

## Verification

1. Original bug no longer reproduces
2. Tests pass: [test command output]
3. No regressions introduced
```

## Git Bisect (When Applicable)

If "it used to work", use bisect:

```bash
git bisect start
git bisect bad HEAD
git bisect good [last-known-good-commit]
# Test each commit, mark good/bad
# Git finds the culprit
git bisect reset
```

## Bug Classification

Categorize for pattern recognition:

| Category        | Indicators                                   |
| --------------- | -------------------------------------------- |
| **Logic**       | Wrong output, off-by-one, edge cases         |
| **State**       | Race conditions, stale data, order-dependent |
| **Resource**    | Leaks, exhaustion, timeouts                  |
| **Integration** | API changes, serialization, version mismatch |
| **Environment** | Works locally, config differences            |
| **Data**        | Bad input, encoding, corruption              |

## Anti-Patterns to Avoid

- **Shotgun debugging**: Random changes hoping something works
- **Debugging by diff**: Blindly reverting without understanding
- **Assumption cascades**: "It must be X" without evidence
- **Fix and forget**: No verification or regression tests

## Output Format

Structure your investigation as:

```markdown
# Debug Session: [Bug Title]

## Reproduction

[Reproducible steps]

## Evidence

[Facts with sources]

## Hypotheses

[Ranked theories]

## Investigation

[Test results for each hypothesis]

## Root Cause

[5 Whys analysis]

## Fix

[Change description with file:line references]

## Verification

[How the fix was validated]

## Prevention

[Regression test or safeguard added]
```

---

**Begin systematic investigation of:** $ARGUMENTS

Start by attempting to reproduce the bug. Do not guess - gather evidence first.
