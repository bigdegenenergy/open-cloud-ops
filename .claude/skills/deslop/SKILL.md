# Deslop Skill

> Aggressive code simplification - ruthlessly eliminate "AI slop" and over-engineering.

## Philosophy

**"Slop"** = Code that works but is:

- Verbose and repetitive
- Over-abstracted (premature DRY)
- Overly defensive (unnecessary null checks everywhere)
- Feature-bloated (YAGNI violations)
- Comment-heavy (explaining bad code instead of fixing it)

**"Deslop"** = Ruthlessly simplify until the code is:

- Minimal but complete
- Self-documenting
- Easy to delete
- Easy to understand at a glance

## The Deslop Mindset

1. **50% Rule**: Can this code be cut in half while preserving behavior?
2. **Delete First**: Before refactoring, ask "can I just delete this?"
3. **Inline Over Abstract**: If an abstraction is used once, inline it
4. **Trust the Runtime**: Don't defensively check what the language guarantees
5. **Comments Are Failures**: If you need a comment, the code isn't clear enough

## Deslop Targets

### Target 1: Unnecessary Abstractions

```python
# SLOP: Over-abstracted for one use case
class UserRepository:
    def __init__(self, db):
        self.db = db

    def get_user_by_id(self, user_id):
        return self.db.query(User).filter_by(id=user_id).first()

# DESLOPPED: Just use the ORM directly
user = db.query(User).filter_by(id=user_id).first()
```

### Target 2: Defensive Overload

```typescript
// SLOP: Paranoid null checks
function getUserName(user: User | null | undefined): string {
  if (user === null || user === undefined) {
    return "";
  }
  if (user.name === null || user.name === undefined) {
    return "";
  }
  return user.name ?? "";
}

// DESLOPPED: Trust the type system
function getUserName(user: User): string {
  return user.name;
}
```

### Target 3: Config-Driven Complexity

```javascript
// SLOP: Configurable everything
const config = {
  maxRetries: process.env.MAX_RETRIES || 3,
  retryDelay: process.env.RETRY_DELAY || 1000,
  enableLogging: process.env.ENABLE_LOGGING === "true",
  logLevel: process.env.LOG_LEVEL || "info",
  // ... 20 more options
};

// DESLOPPED: Hard-code until you actually need flexibility
const MAX_RETRIES = 3;
const RETRY_DELAY = 1000;
```

### Target 4: Dead Code & Unused Exports

```python
# SLOP: "Might need this later"
def legacy_handler():  # TODO: Remove after migration
    pass

def unused_utility():  # Added for potential future use
    pass

# DESLOPPED: Delete it. Git remembers.
# (nothing here - it's gone)
```

### Target 5: Comment Clutter

```javascript
// SLOP: Comments explaining obvious code
// Get the user's age
const age = user.age;

// Check if user is adult
if (age >= 18) {
  // User is an adult, allow access
  allowAccess();
}

// DESLOPPED: Self-documenting code needs no comments
if (user.age >= 18) {
  allowAccess();
}
```

## Deslop Protocol

### Step 1: Measure Before

```bash
# Count lines, functions, files
wc -l src/**/*.ts
grep -c "function\|const.*=.*=>" src/**/*.ts
```

### Step 2: Identify Slop Patterns

Look for:

- [ ] Functions < 5 lines called only once → inline
- [ ] Classes with single methods → use plain functions
- [ ] Interfaces implemented once → delete the interface
- [ ] Try/catch that just re-throws → remove it
- [ ] Async/await on synchronous code → remove it
- [ ] Lodash for native operations → use native
- [ ] Type assertions that narrow to the same type → remove

### Step 3: Apply Surgical Cuts

**Order of operations:**

1. Delete dead code (unused exports, unreachable branches)
2. Inline single-use abstractions
3. Remove unnecessary null checks
4. Simplify conditionals
5. Flatten nested structures

### Step 4: Verify

```bash
# Run tests after EVERY change
npm test  # or pytest, cargo test, etc.
```

### Step 5: Measure After

Report the diff:

- Lines removed: X
- Functions eliminated: Y
- Files deleted: Z
- Test coverage: unchanged or improved

## Deslop Heuristics

### When to Inline

| Pattern                                  | Action             |
| ---------------------------------------- | ------------------ |
| Function called once                     | Inline it          |
| Function < 3 lines                       | Probably inline it |
| "Helper" with 1 caller                   | Inline it          |
| Wrapper that just calls another function | Delete it          |

### When to Delete

| Pattern                  | Action                    |
| ------------------------ | ------------------------- |
| Commented-out code       | Delete (git remembers)    |
| TODO older than 3 months | Delete or do it now       |
| "Might need later" code  | Delete (YAGNI)            |
| Unused imports           | Delete                    |
| Empty catch blocks       | Delete or handle properly |

### When NOT to Deslop

- **Public APIs**: Breaking changes affect users
- **Performance-critical paths**: Measure before simplifying
- **Security code**: Defensive checks may be intentional
- **Compliance requirements**: Some verbosity is legally required

## Deslop Metrics

Track these before/after:

```
METRIC              BEFORE    AFTER     CHANGE
Lines of code       1,247     823       -34%
Functions           89        52        -42%
Files               24        18        -25%
Avg function length 14 loc    9 loc     -36%
Cyclomatic complexity 4.2     2.8       -33%
Test coverage       78%       82%       +4%
```

## Anti-Patterns to Watch

### False Simplification

Don't sacrifice correctness for brevity:

```python
# BAD deslop: Lost error handling
data = json.loads(response.text)

# GOOD: Keep essential error handling
try:
    data = json.loads(response.text)
except json.JSONDecodeError:
    raise InvalidResponseError(response.text)
```

### Over-Inlining

Don't inline if it hurts readability:

```python
# BAD deslop: Unreadable one-liner
users = [u for u in [db.get(id) for id in ids if id] if u and u.active and u.verified and not u.banned]

# GOOD: Readable pipeline
users = [db.get(id) for id in ids if id]
users = [u for u in users if u and u.active and u.verified and not u.banned]
```

## Activation Triggers

This skill auto-activates when prompts contain:

- "deslop", "remove slop", "clean up AI code"
- "aggressive refactor", "simplify drastically"
- "cut code", "reduce complexity"
- "over-engineered", "too verbose"

## Integration with Other Skills

- **refactoring**: Deslop is refactoring's aggressive cousin
- **testing-patterns**: Always verify with tests after deslop
- **debugging**: Simpler code = easier debugging
