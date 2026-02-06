# Zeno: Surgical Code Analysis

You are performing **surgical code analysis** with mandatory evidence citations.

## Target

$ARGUMENTS

If no target specified, analyze recent changes:

```bash
git diff --name-only HEAD~5
```

## Core Rule

**Every finding MUST include a file:line citation with code snippet.**

No citation = Finding rejected.

## Analysis Protocol

### Step 1: Scope Identification

Determine what to analyze:

- **Specific files**: Analyze provided paths
- **Directory**: Scan all source files in directory
- **Git diff**: Focus on changed lines only
- **Pattern**: Find files matching glob

```bash
# Get file list
git diff --name-only HEAD~5 2>/dev/null || find . -name "*.ts" -o -name "*.py" -o -name "*.js" | head -50
```

### Step 2: Pattern Search

Search for common vulnerability patterns:

```bash
# Security patterns
grep -rn "eval\|exec\|Function(" --include="*.ts" --include="*.js" .
grep -rn "innerHTML\|outerHTML\|document.write" --include="*.ts" --include="*.js" .
grep -rn "password.*=\|secret.*=\|api_key.*=" .

# SQL patterns
grep -rn "execute.*f\"\|execute.*%\|+ .*query" --include="*.py" .
grep -rn "query.*\`.*\$\{" --include="*.ts" --include="*.js" .

# Error handling
grep -rn "catch.*{\s*}" --include="*.ts" --include="*.js" .
```

### Step 3: Detailed Analysis

For each potential issue:

1. **Read** the file with line numbers
2. **Verify** the issue exists (not false positive)
3. **Check** for mitigating controls
4. **Document** with full evidence

### Step 4: Generate Report

Output as structured JSON:

```json
{
  "analysis": {
    "target": "<what was analyzed>",
    "files_scanned": 15,
    "timestamp": "2026-01-25T10:30:00Z"
  },
  "summary": {
    "total": 5,
    "critical": 1,
    "high": 2,
    "medium": 1,
    "low": 1
  },
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
      "explanation": "User input directly interpolated into SQL query",
      "recommendation": "Use parameterized queries",
      "references": ["CWE-89"]
    }
  ]
}
```

## Severity Classification

| Severity     | Criteria                 | Examples                             |
| ------------ | ------------------------ | ------------------------------------ |
| **Critical** | Exploitable, high impact | SQL injection, RCE, auth bypass      |
| **High**     | Significant risk         | XSS, SSRF, hardcoded secrets         |
| **Medium**   | Moderate risk            | Missing validation, info disclosure  |
| **Low**      | Minor concern            | Code smell, optimization opportunity |

## Category IDs

Use these prefixes for finding IDs:

- `SEC-XXX`: Security vulnerabilities
- `COR-XXX`: Correctness/logic errors
- `PERF-XXX`: Performance issues
- `MAINT-XXX`: Maintainability concerns

## False Positive Prevention

Before reporting, verify:

1. **Reachability**: Is this code actually executed?
2. **Mitigation**: Is there validation elsewhere?
3. **Context**: Is this test/mock code?
4. **Types**: Does the type system prevent this?

If uncertain, add `"confidence": "medium"` to the finding.

## Output Requirements

1. **JSON report** with all findings
2. **Summary counts** by severity and category
3. **Actionable recommendations** for each finding
4. **File:line citations** for every issue

## Example Output

````
## Zeno Analysis Report

**Target:** src/api/
**Files Scanned:** 23
**Findings:** 4 (1 critical, 2 high, 1 medium)

---

### SEC-001: SQL Injection [CRITICAL]

**Location:** `src/db/users.py:42`

```python
cursor.execute(f"SELECT * FROM users WHERE id={user_id}")
````

**Issue:** User-controlled `user_id` interpolated directly into SQL.

**Fix:** Use parameterized query:

```python
cursor.execute("SELECT * FROM users WHERE id=?", (user_id,))
```

**References:** CWE-89, OWASP A03:2021

---

### SEC-002: Hardcoded API Key [HIGH]

**Location:** `src/config/settings.ts:15`

```typescript
const API_KEY = "sk-live-abc123xyz789";
```

**Issue:** Production API key committed to source code.

**Fix:** Use environment variable:

```typescript
const API_KEY = process.env.API_KEY;
```

---

<full JSON report follows>
```

---

**Begin surgical analysis of:** $ARGUMENTS
