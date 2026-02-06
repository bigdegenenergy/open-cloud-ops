# Systematic Debugging Skill

> Scientific method applied to bug hunting - no guessing, only evidence.

## Core Principle

**Never guess. Always test.**

Random code changes hoping to fix a bug are debugging theater. Systematic debugging requires:

1. Clear hypothesis
2. Testable prediction
3. Controlled experiment
4. Evidence-based conclusion

## The Debugging Protocol

### Step 1: Reproduce (CRITICAL)

**No reproduction = No debugging**

```markdown
## Reproduction Case

**Steps:**

1. [Exact step 1]
2. [Exact step 2]
3. [Exact step 3]

**Expected:** [What should happen]
**Actual:** [What actually happens]

**Reproducibility:** [Always / Sometimes / Rarely]
**Environment:** [OS, versions, config]
```

If you can't reproduce, you can't debug. Stop and get better reproduction steps.

### Step 2: Gather Evidence

Before forming hypotheses, collect facts:

```markdown
## Evidence Log

| #   | Timestamp | Observation                         | Source            |
| --- | --------- | ----------------------------------- | ----------------- |
| E1  | 10:32     | Error: "Connection refused"         | server.log:1247   |
| E2  | 10:32     | Last successful request at 10:31:58 | access.log        |
| E3  | 10:33     | Memory usage at 94%                 | metrics dashboard |
| E4  | 10:33     | No recent deployments               | deploy.log        |
```

Evidence sources:

- [ ] Error messages and stack traces
- [ ] Application logs
- [ ] System metrics (CPU, memory, disk, network)
- [ ] Recent changes (git log, deploy history)
- [ ] User reports
- [ ] Monitoring alerts

### Step 3: Form Hypotheses

Based on evidence, generate ranked hypotheses:

```markdown
## Hypotheses

| ID  | Hypothesis                         | Evidence Supporting                       | Likelihood |
| --- | ---------------------------------- | ----------------------------------------- | ---------- |
| H1  | Database connection pool exhausted | E1 (connection refused), E3 (high memory) | High       |
| H2  | Memory leak causing OOM            | E3 (94% memory)                           | Medium     |
| H3  | Network partition                  | E1 (connection refused)                   | Low        |
```

**Rules for good hypotheses:**

- Must be falsifiable
- Must explain ALL observed symptoms
- Simpler explanations preferred (Occam's Razor)

### Step 4: Test Hypotheses

For each hypothesis, starting with highest likelihood:

```markdown
## Hypothesis Test: H1 - Connection Pool Exhausted

**Prediction:** If H1 is true, then:

- Connection pool metrics will show 0 available connections
- Increasing pool size should resolve the issue temporarily

**Test:**

1. Check pool metrics: `SELECT * FROM pg_stat_activity;`
2. Count active connections vs pool max

**Result:**

- Active connections: 47
- Pool max: 50
- Available: 3

**Conclusion:** REJECTED - Pool is not exhausted (3 available)

---

## Hypothesis Test: H2 - Memory Leak

**Prediction:** If H2 is true, then:

- Memory should increase over time with stable traffic
- Specific process should show growing RSS

**Test:**

1. Monitor memory over 10 minutes: `watch -n 5 'ps aux --sort=-rss | head'`
2. Check for growth pattern

**Result:**

- Node process RSS: 1.2GB → 1.4GB → 1.6GB (10 min)
- Traffic: stable at ~100 req/s

**Conclusion:** CONFIRMED - Memory growing without proportional traffic increase
```

### Step 5: Root Cause Analysis

Once you've confirmed a hypothesis, dig deeper:

```markdown
## Root Cause Analysis

**Confirmed Issue:** Memory leak in Node.js process

**5 Whys:**

1. Why is memory growing? → Objects not being garbage collected
2. Why aren't they GC'd? → References held in global cache
3. Why are references held? → Cache never evicts entries
4. Why no eviction? → TTL was set to 0 (infinite) by mistake
5. Why was TTL 0? → Default value, never configured

**Root Cause:** Cache TTL defaulting to 0 (infinite retention)

**Fix:** Set explicit TTL: `cache.set(key, value, { ttl: 3600 })`
```

### Step 6: Verify Fix

```markdown
## Fix Verification

**Change:** Set cache TTL to 3600 seconds

**Test Plan:**

1. Deploy fix to staging
2. Run load test (100 req/s for 30 min)
3. Monitor memory usage

**Results:**

- Memory: Stable at ~800MB (no growth)
- Response times: Unchanged
- Error rate: 0%

**Conclusion:** Fix verified. Ready for production.
```

## Git Bisect Protocol

When you know "it worked before", use bisect:

```bash
# Start bisect
git bisect start

# Mark current (broken) state
git bisect bad HEAD

# Mark last known good state
git bisect good v1.2.0

# Git will checkout a middle commit
# Test it, then mark:
git bisect good  # if it works
git bisect bad   # if it's broken

# Repeat until Git finds the culprit
# Result: "abc123 is the first bad commit"

# Clean up
git bisect reset
```

**Bisect log template:**

```markdown
## Git Bisect Log

**Symptom:** [What's broken]
**Good commit:** v1.2.0 (2026-01-01)
**Bad commit:** HEAD (2026-01-25)

| Commit | Date  | Result | Notes              |
| ------ | ----- | ------ | ------------------ |
| abc123 | 01-15 | BAD    | Error reproduces   |
| def456 | 01-08 | GOOD   | Works correctly    |
| ghi789 | 01-12 | BAD    | Error reproduces   |
| jkl012 | 01-10 | GOOD   | Works correctly    |
| mno345 | 01-11 | BAD    | ← First bad commit |

**Culprit:** mno345 - "Refactor cache layer"
**Author:** developer@example.com
**Files Changed:** src/cache/index.ts
```

## Bug Classification Taxonomy

Categorize bugs to find patterns:

| Category        | Examples                      | Common Causes                     |
| --------------- | ----------------------------- | --------------------------------- |
| **Logic**       | Wrong calculation, off-by-one | Incorrect algorithm, edge cases   |
| **State**       | Race condition, stale data    | Async issues, missing locks       |
| **Resource**    | Leak, exhaustion, deadlock    | Missing cleanup, unbounded growth |
| **Integration** | API mismatch, serialization   | Contract changes, version skew    |
| **Environment** | Works locally, fails in prod  | Config diff, missing deps         |
| **Data**        | Corrupt input, encoding       | Validation gaps, assumptions      |

## Debugging Tools Quick Reference

### JavaScript/TypeScript

```javascript
// Conditional breakpoint
debugger;

// Structured logging
console.log(JSON.stringify({ event: "debug", data }, null, 2));

// Performance timing
console.time("operation");
// ... code ...
console.timeEnd("operation");

// Memory snapshot
if (global.gc) global.gc();
console.log(process.memoryUsage());
```

### Python

```python
# Drop into debugger
import pdb; pdb.set_trace()  # or: breakpoint()

# Structured logging
import logging
logging.debug(f"State: {vars(obj)}")

# Memory profiling
import tracemalloc
tracemalloc.start()
# ... code ...
snapshot = tracemalloc.take_snapshot()
top_stats = snapshot.statistics('lineno')[:10]
```

### General

```bash
# Trace system calls
strace -f -e trace=network ./app

# Watch file changes
inotifywait -m -r ./src

# Network debugging
tcpdump -i any port 8080 -w capture.pcap

# Process state
lsof -p $PID
```

## Anti-Patterns

### Shotgun Debugging

**Wrong:** "Let me try changing this... and this... and this..."
**Right:** Form hypothesis → Test → Conclude → Repeat

### Debugging by Diff

**Wrong:** "It worked yesterday, let me just revert everything"
**Right:** Use git bisect to find the exact breaking change

### Assumption Cascades

**Wrong:** "It must be the database" (without evidence)
**Right:** Gather evidence FIRST, then form hypotheses

### Fix-and-Forget

**Wrong:** Apply fix, close ticket, move on
**Right:** Verify fix, add regression test, document root cause

## Activation Triggers

This skill auto-activates when prompts contain:

- "systematic debug", "methodical debug"
- "root cause analysis", "RCA"
- "bisect", "git bisect"
- "why is this happening", "investigate bug"
- "reproduce", "intermittent"

## Output Template

When debugging, always structure output as:

```markdown
# Debug Session: [Bug Title]

## Reproduction

[Steps to reproduce]

## Evidence

[Collected facts with sources]

## Hypotheses

[Ranked list with likelihood]

## Tests

[For each hypothesis: prediction, test, result, conclusion]

## Root Cause

[5 Whys analysis]

## Fix

[Change description]

## Verification

[How fix was validated]

## Prevention

[How to prevent similar bugs]
```
