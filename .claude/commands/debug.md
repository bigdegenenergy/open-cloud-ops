# Debug Investigation

Investigate and diagnose issues systematically.

## Investigation Process

1. **Reproduce the Issue**
   - Understand the reported problem
   - Identify steps to reproduce
   - Note expected vs actual behavior

2. **Gather Evidence**
   - Search for error messages in logs
   - Find related code paths
   - Check recent changes (git log)
   - Look for similar past issues

3. **Form Hypotheses**
   - List potential root causes
   - Rank by likelihood
   - Identify what evidence would confirm/deny each

4. **Test Hypotheses**
   - Add strategic logging if needed
   - Trace data flow through the system
   - Check edge cases and boundary conditions

5. **Identify Root Cause**
   - Pinpoint the exact source of the bug
   - Understand WHY it's happening
   - Note any contributing factors

6. **Propose Fix**
   - Suggest minimal targeted fix
   - Consider side effects
   - Note if tests need to be added/updated

## Output Format

````
## Debug Investigation

### Problem Description
[What's happening vs what should happen]

### Reproduction
[Steps to reproduce or conditions that trigger the issue]

### Investigation Trail

1. **Hypothesis**: [What might be wrong]
   - **Evidence**: [What I found]
   - **Status**: Confirmed / Ruled Out

2. **Hypothesis**: [Next theory]
   ...

### Root Cause
**Location**: `file:line`
**Issue**: [Specific description]
**Why**: [Underlying reason]

### Proposed Fix
```[language]
// Suggested code change
````

### Impact Assessment

- **Risk**: Low/Medium/High
- **Files Affected**: [list]
- **Tests Needed**: [list]

### Additional Recommendations

- [Any related issues to address]
- [Preventive measures]

```

Focus on finding the root cause, not just treating symptoms. Be methodical and evidence-based.
```
