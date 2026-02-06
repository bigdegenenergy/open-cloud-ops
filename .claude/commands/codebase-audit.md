---
description: Comprehensive codebase audit. Reviews code quality, security, performance, and architecture.
---

# Codebase Audit Workflow

You are orchestrating a comprehensive codebase audit that examines multiple dimensions of code quality.

## Audit Dimensions

| Dimension | Agent | Focus Areas |
|-----------|-------|-------------|
| Security | @security-auditor | OWASP Top 10, secrets, dependencies |
| Architecture | @backend-architect | Design patterns, coupling, modularity |
| Code Quality | @code-reviewer | Readability, maintainability, standards |
| Performance | @performance-analyzer | Bottlenecks, efficiency, scaling |
| Testing | @test-automator | Coverage, quality, reliability |

## Phase 1: Security Audit

```
Invoke @security-auditor to scan for:
- Injection vulnerabilities (SQL, XSS, command)
- Authentication/authorization issues
- Hardcoded secrets and credentials
- Dependency vulnerabilities
- Security misconfigurations
```

## Phase 2: Architecture Review

```
Invoke @backend-architect to review:
- Overall system design
- Service boundaries and coupling
- API design consistency
- Data flow and dependencies
- Technical debt assessment
```

## Phase 3: Code Quality Review

```
Invoke @code-reviewer to assess:
- Code readability and clarity
- Naming conventions
- Error handling patterns
- Documentation quality
- Code duplication
```

## Phase 4: Performance Analysis

```
Invoke @performance-analyzer to check:
- Database query efficiency
- Memory usage patterns
- I/O bottlenecks
- Caching opportunities
- Resource utilization
```

## Phase 5: Test Coverage Review

```
Invoke @test-automator to evaluate:
- Unit test coverage
- Integration test coverage
- Test quality and reliability
- Missing test scenarios
- Flaky test identification
```

## Audit Report Template

```markdown
# Codebase Audit Report

## Executive Summary
- **Audit Date**: YYYY-MM-DD
- **Scope**: [What was audited]
- **Overall Health**: [Healthy/Needs Attention/Critical]

## Findings Summary

| Category | Critical | High | Medium | Low |
|----------|----------|------|--------|-----|
| Security | X | X | X | X |
| Architecture | X | X | X | X |
| Code Quality | X | X | X | X |
| Performance | X | X | X | X |
| Testing | X | X | X | X |

## Detailed Findings

### Security
[Findings from security audit]

### Architecture
[Findings from architecture review]

### Code Quality
[Findings from code review]

### Performance
[Findings from performance analysis]

### Testing
[Findings from test review]

## Recommendations

### Immediate Actions (This Sprint)
1. [Critical security fixes]
2. [High-priority issues]

### Short-Term (Next 2-4 Weeks)
1. [Architectural improvements]
2. [Performance optimizations]

### Long-Term (Next Quarter)
1. [Technical debt reduction]
2. [Test coverage improvement]

## Metrics

- Lines of Code: X
- Test Coverage: X%
- Dependency Count: X
- Open Security Issues: X
- Technical Debt Hours: X
```

## Current Codebase Info

**Project Structure:**
```bash
!`find . -type f -name "*.py" -o -name "*.ts" -o -name "*.js" -o -name "*.go" 2>/dev/null | head -20 || echo "Unable to list files"`
```

**Git Stats:**
```bash
!`git log --oneline -10 2>/dev/null || echo "Not a git repository"`
```

## Instructions

1. Start with security audit (highest priority)
2. Review architecture for systemic issues
3. Assess code quality for maintainability
4. Analyze performance for scalability
5. Evaluate testing for reliability

After each phase, document findings and prioritize by severity.

**Important**: Focus on actionable findings. Don't report every minor style issue - prioritize what matters for the codebase's health.
