# Surgical Analysis Skill

> Evidence-based code analysis - every finding must cite specific file:line references.

## Core Principle

**"Lost in the Middle" Problem Solved**

Traditional code review dumps entire files into context, losing precision. Surgical analysis uses targeted retrieval:

1. **Grep** to find patterns
2. **Read** with line numbers for context
3. **Cite** specific lines as evidence

This solves the context window problem while maintaining accuracy.

## Citation Format

Every finding MUST include citations in this format:

```
path/to/file.ext:LINE
path/to/file.ext:START-END
```

### Good Citation Example

````markdown
**Finding:** SQL Injection vulnerability

**Evidence:** `src/db/users.py:42`

```python
cursor.execute(f"SELECT * FROM users WHERE id={user_id}")
```
````

**Explanation:** User-controlled `user_id` is interpolated directly into SQL query.

**Recommendation:** Use parameterized query:

```python
cursor.execute("SELECT * FROM users WHERE id=?", (user_id,))
```

````

### Bad Citation Example (DO NOT DO THIS)

```markdown
**Finding:** There might be a security issue somewhere in the auth module.

**Evidence:** I noticed some concerning patterns.

**Recommendation:** Review the code more carefully.
````

## Evidence Requirements

For each finding, provide:

| Field            | Required    | Description                  |
| ---------------- | ----------- | ---------------------------- |
| `file`           | Yes         | Exact file path              |
| `line_start`     | Yes         | First line of issue          |
| `line_end`       | Yes         | Last line of issue           |
| `snippet`        | Yes         | Actual code (verbatim)       |
| `context_before` | Recommended | 1-2 lines before for context |
| `context_after`  | Recommended | 1-2 lines after for context  |

## Analysis Patterns

### Pattern 1: Security Scan

```bash
# Find potential injection points
grep -rn "execute.*\$\|execute.*f\"|\.format(" src/

# Find hardcoded secrets
grep -rn "password.*=.*['\"]|api_key.*=.*['\"]|secret.*=.*['\"]" src/

# Find eval/exec usage
grep -rn "eval(|exec(|Function(" src/
```

### Pattern 2: Error Handling Audit

```bash
# Find empty catch blocks
grep -rn "catch.*{[\s]*}" src/

# Find swallowed errors
grep -rn "catch.*{.*console\." src/

# Find missing error handling
grep -rn "await.*[^try]" src/  # async without try
```

### Pattern 3: Performance Issues

```bash
# Find N+1 query patterns (loop with DB call)
grep -rn "for.*{" -A5 src/ | grep -E "query|find|get|fetch"

# Find synchronous file operations
grep -rn "readFileSync|writeFileSync" src/

# Find unbounded collections
grep -rn "\.push\(|\.concat\(" src/
```

### Pattern 4: Complexity Analysis

```bash
# Find deeply nested code (4+ levels)
grep -rn "if.*{" src/ | grep -E "^\s{16,}"

# Find long functions (50+ lines)
# Use AST tools for accurate measurement

# Find duplicated code blocks
# Use similarity detection tools
```

## Cross-Reference Protocol

When analyzing, cross-reference findings:

```markdown
## Finding SEC-001: Unsanitized Input

**Primary:** `src/api/users.ts:42`
**Related:**

- Input source: `src/routes/users.ts:15` (where user_id comes from)
- Similar pattern: `src/api/products.ts:67` (same vulnerability)
```

## Confidence Levels

Mark findings with confidence:

| Level      | When to Use                       |
| ---------- | --------------------------------- |
| **High**   | Clear evidence, definite issue    |
| **Medium** | Likely issue, needs verification  |
| **Low**    | Possible issue, context-dependent |

```json
{
  "id": "SEC-001",
  "confidence": "high",
  "reason": "Direct string interpolation in SQL with user input"
}
```

## Verification Checklist

Before finalizing a finding:

- [ ] Read the actual lines (not just grep output)
- [ ] Check for validation in calling code
- [ ] Verify the code is reachable
- [ ] Check if there's a security wrapper
- [ ] Look for compensating controls
- [ ] Confirm it's not test/mock code

## False Positive Patterns

### Common False Positives

| Pattern                   | Why It's Often False                  |
| ------------------------- | ------------------------------------- |
| `eval()` in build tools   | Build-time only, not runtime          |
| SQL in migration files    | Static schema, no user input          |
| Hardcoded values in tests | Test fixtures, not production         |
| `any` type in .d.ts       | Type declarations, not implementation |

### Mitigation Check

Before reporting, verify no mitigation exists:

```javascript
// This LOOKS vulnerable:
const query = `SELECT * FROM users WHERE id = ${id}`;

// But check the caller:
function getUser(id: number) {  // Type ensures it's a number
  // ...
}
```

## Output Formats

### JSON (Machine-Readable)

```json
{
  "findings": [
    {
      "id": "SEC-001",
      "severity": "critical",
      "category": "security",
      "title": "SQL Injection",
      "evidence": {
        "file": "src/db/users.py",
        "line_start": 42,
        "line_end": 42,
        "snippet": "cursor.execute(f\"SELECT * FROM users WHERE id={user_id}\")"
      },
      "explanation": "...",
      "recommendation": "..."
    }
  ]
}
```

### Markdown (Human-Readable)

````markdown
## SEC-001: SQL Injection [CRITICAL]

**Location:** `src/db/users.py:42`

**Code:**

```python
cursor.execute(f"SELECT * FROM users WHERE id={user_id}")
```
````

**Issue:** User-controlled input directly in SQL query.

**Fix:**

```python
cursor.execute("SELECT * FROM users WHERE id=?", (user_id,))
```

```

## Activation Triggers

This skill auto-activates when prompts contain:
- "analyze", "audit", "review code"
- "find vulnerabilities", "security scan"
- "evidence-based", "cite lines"
- "surgical", "precise analysis"

## Integration

Works with:
- **@zeno-analyzer**: Agent that implements this methodology
- **@code-reviewer**: Provides evidence for review decisions
- **@security-auditor**: Deeper security-specific analysis
- **/zeno**: Command that invokes surgical analysis
```
