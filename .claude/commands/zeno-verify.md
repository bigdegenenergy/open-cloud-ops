# Zeno Verify: Citation Validation

Validate that Zeno analysis citations still match the current codebase.

## Purpose

After code changes, previous Zeno findings may become stale:

- Line numbers shifted
- Code was refactored
- Issue was fixed
- File was renamed/deleted

This command verifies citations are still valid.

## Input

$ARGUMENTS

Accepts either:

1. **JSON file path**: Path to a Zeno report JSON file
2. **Inline findings**: Pasted JSON findings array
3. **Finding IDs**: Specific finding IDs to verify (e.g., "SEC-001 SEC-002")

## Verification Protocol

### Step 1: Parse Findings

Extract citations from the input:

```json
{
  "findings": [
    {
      "id": "SEC-001",
      "evidence": {
        "file": "src/db/users.py",
        "line_start": 42,
        "line_end": 42,
        "snippet": "cursor.execute(f\"SELECT * FROM users WHERE id={user_id}\")"
      }
    }
  ]
}
```

### Step 2: Verify Each Citation

For each finding, check:

1. **File Exists**

```bash
test -f "src/db/users.py" && echo "EXISTS" || echo "MISSING"
```

2. **Line Number Valid**

```bash
sed -n '42p' src/db/users.py
```

3. **Snippet Matches**
   Compare the stored snippet with current code at that line.

### Step 3: Classification

| Status     | Meaning                           |
| ---------- | --------------------------------- |
| `VALID`    | Citation matches current code     |
| `SHIFTED`  | Code exists but at different line |
| `MODIFIED` | Line exists but code differs      |
| `FIXED`    | Vulnerability no longer present   |
| `MISSING`  | File or line no longer exists     |

### Step 4: Generate Report

````markdown
## Zeno Verification Report

**Findings Verified:** 5
**Valid:** 3
**Stale:** 2

---

### SEC-001: VALID

**Original:** `src/db/users.py:42`
**Current:** Same location, same code
**Status:** Issue still present

---

### SEC-002: SHIFTED

**Original:** `src/api/auth.ts:67`
**Current:** `src/api/auth.ts:72` (+5 lines)
**Cause:** Code added above
**Status:** Issue still present at new location

---

### SEC-003: FIXED

**Original:** `src/config/settings.ts:15`
**Current:** Code changed
**Before:**

```typescript
const API_KEY = "sk-live-abc123xyz789";
```
````

**After:**

```typescript
const API_KEY = process.env.API_KEY;
```

**Status:** Issue resolved

---

### SEC-004: MISSING

**Original:** `src/legacy/handler.py:89`
**Current:** File deleted
**Status:** Finding obsolete (remove from report)

````

## Shift Detection

When code has shifted, attempt to locate it:

```bash
# Search for the snippet in the file
grep -n "cursor.execute" src/db/users.py
````

If found at a different line, report the new location.

## Fix Detection

Compare current code against the recommendation:

```python
# Original (vulnerable)
cursor.execute(f"SELECT * FROM users WHERE id={user_id}")

# Recommended (fixed)
cursor.execute("SELECT * FROM users WHERE id=?", (user_id,))

# Current code
cursor.execute("SELECT * FROM users WHERE id=?", (user_id,))

# Result: FIXED (current matches recommendation)
```

## Output Format

```json
{
  "verification": {
    "timestamp": "2026-01-25T11:00:00Z",
    "findings_checked": 5,
    "results": {
      "valid": 3,
      "shifted": 1,
      "fixed": 1,
      "missing": 0
    }
  },
  "findings": [
    {
      "id": "SEC-001",
      "status": "valid",
      "original_location": "src/db/users.py:42",
      "current_location": "src/db/users.py:42",
      "notes": "Issue still present"
    },
    {
      "id": "SEC-002",
      "status": "shifted",
      "original_location": "src/api/auth.ts:67",
      "current_location": "src/api/auth.ts:72",
      "shift": "+5",
      "notes": "Code moved due to additions above"
    },
    {
      "id": "SEC-003",
      "status": "fixed",
      "original_location": "src/config/settings.ts:15",
      "current_location": "src/config/settings.ts:15",
      "notes": "Now uses environment variable"
    }
  ],
  "actions_needed": [
    "Update SEC-002 citation to line 72",
    "Remove SEC-003 from active findings (fixed)"
  ]
}
```

## Usage Examples

```bash
# Verify all findings in a report
/zeno-verify .claude/artifacts/zeno-report.json

# Verify specific findings
/zeno-verify SEC-001 SEC-002 SEC-003

# Verify after making fixes
/zeno-verify --check-fixes .claude/artifacts/zeno-report.json
```

## Integration

Use after:

- Running `/zeno` to generate initial report
- Making code changes based on findings
- Before presenting findings to stakeholders

---

**Verify citations for:** $ARGUMENTS
