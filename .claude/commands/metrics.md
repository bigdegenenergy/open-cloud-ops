---
description: Generate team productivity and code quality metrics report.
model: haiku
allowed-tools: Bash(git*), Bash(npm*), Read(*), Grep(*), Glob(*)
---

# Team Metrics Report

You are the **Metrics Analyst** responsible for tracking team productivity and code quality.

## Context

- **Repository:** !`basename $(git rev-parse --show-toplevel)`
- **Current Branch:** !`git branch --show-current`
- **Last Week Commits:** !`git log --oneline --since="1 week ago" | wc -l`
- **Contributors:** !`git shortlog -sn --since="1 week ago" | head -5`

## Metrics to Gather

### 1. Code Velocity

```bash
# Commits in last 7 days
git log --oneline --since="7 days ago" | wc -l

# Lines changed
git diff --stat $(git log --since="7 days ago" --format=%H | tail -1)..HEAD

# Files modified
git diff --name-only $(git log --since="7 days ago" --format=%H | tail -1)..HEAD | wc -l
```

### 2. Code Quality

```bash
# Test coverage (if available)
npm run test:coverage 2>/dev/null || echo "No coverage script"

# Linting issues
npm run lint 2>&1 | tail -5

# Type errors
npm run type-check 2>&1 | grep -c "error" || echo "0"
```

### 3. Git Health

```bash
# Branches count
git branch -r | wc -l

# Stale branches (no commits in 30 days)
git for-each-ref --sort=-committerdate refs/remotes/ --format='%(committerdate:short) %(refname:short)' | head -10

# Uncommitted changes
git status --porcelain | wc -l
```

### 4. Dependency Health

```bash
# Outdated packages
npm outdated 2>/dev/null | wc -l || echo "N/A"

# Security vulnerabilities
npm audit 2>/dev/null | grep -E "vulnerabilities|found" || echo "N/A"
```

## Report Format

```markdown
# Weekly Metrics Report

**Period:** [date range]
**Repository:** [repo name]

## Velocity

- Commits: X (trend: +/-Y%)
- Lines changed: +X / -Y
- Files modified: Z
- Contributors: N active

## Code Quality

- Test coverage: X% (target: 80%)
- Linting issues: Y (trend: +/-Z)
- Type errors: N

## Git Health

- Active branches: X
- Stale branches: Y (>30 days)
- Open PRs: Z

## Dependencies

- Outdated packages: X
- Security vulnerabilities: Y (critical: Z)

## Recommendations

1. [Action item based on metrics]
2. [Action item based on metrics]
3. [Action item based on metrics]

## Trends

[Compare to previous week if data available]
```

## Output

Generate a comprehensive report with:

1. Raw metrics with context
2. Comparison to targets/baselines
3. Trend indicators (improving/declining)
4. Specific recommendations
5. Action items for next week

**Your goal: Data-driven insights to improve team productivity.**
