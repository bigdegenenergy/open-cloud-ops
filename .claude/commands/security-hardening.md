---
description: Security hardening workflow. Coordinates security audit, vulnerability assessment, and remediation.
---

# Security Hardening Workflow

You are orchestrating a comprehensive security hardening workflow that coordinates multiple agents.

## Workflow Overview

This workflow performs a thorough security assessment and guides remediation:

1. **Vulnerability Scanning** - Identify security issues
2. **Risk Assessment** - Prioritize findings
3. **Remediation** - Fix identified issues
4. **Verification** - Confirm fixes are effective

## Phase 1: Security Audit

First, invoke the security auditor for a comprehensive scan:

```
Invoke @security-auditor to perform a full security audit covering:
- OWASP Top 10 vulnerabilities
- Dependency vulnerabilities (npm/pip/cargo audit)
- Hardcoded secrets detection
- Authentication/authorization review
- Input validation checks
```

## Phase 2: Risk Assessment

Categorize findings by severity:

| Severity | Response Time | Examples |
|----------|---------------|----------|
| CRITICAL | Immediate | RCE, SQL injection, auth bypass |
| HIGH | 24 hours | XSS, IDOR, sensitive data exposure |
| MEDIUM | 1 week | Missing security headers, weak crypto |
| LOW | Next sprint | Minor info disclosure, verbose errors |

## Phase 3: Remediation

For each finding, use appropriate agents:

### Code Vulnerabilities
```
Invoke @backend-architect for secure design patterns
Invoke @python-pro or @typescript-pro for secure implementation
```

### Infrastructure Vulnerabilities
```
Invoke @kubernetes-architect for K8s security hardening
Invoke @infrastructure-engineer for infrastructure fixes
```

### Dependency Vulnerabilities
```
Update vulnerable dependencies
Apply security patches
Consider alternatives for unmaintained packages
```

## Phase 4: Verification

After remediation:
```
Invoke @security-auditor to verify fixes are effective
Invoke @test-automator to add security regression tests
```

## Security Checklist

### Authentication
- [ ] Strong password policies
- [ ] MFA available for sensitive operations
- [ ] Secure session management
- [ ] Rate limiting on auth endpoints
- [ ] Account lockout after failed attempts

### Authorization
- [ ] Principle of least privilege
- [ ] RBAC or ABAC implemented
- [ ] Authorization on every endpoint
- [ ] No horizontal privilege escalation

### Data Protection
- [ ] Encryption at rest
- [ ] Encryption in transit (TLS 1.3)
- [ ] Sensitive data masked in logs
- [ ] PII handling compliance

### Infrastructure
- [ ] Security groups restrictive
- [ ] Network segmentation
- [ ] Secrets in vault (not env vars)
- [ ] Regular patching process

## Current State

**Recent Commits:** !`git log --oneline -5`

**Dependency Status:**
```bash
# Check for known vulnerabilities
npm audit 2>/dev/null || pip-audit 2>/dev/null || echo "Run security audit manually"
```

## Report Template

Document findings in this format:

```markdown
# Security Audit Report

## Executive Summary
- Audit Date: YYYY-MM-DD
- Overall Risk: [CRITICAL/HIGH/MEDIUM/LOW]
- Findings: X Critical, Y High, Z Medium

## Findings

### CRITICAL-001: [Title]
- Location: file:line
- Description: ...
- Impact: ...
- Remediation: ...
- Status: [Open/Fixed/Accepted Risk]
```

**Important**: Never ignore CRITICAL or HIGH severity findings. They must be fixed or have explicit risk acceptance from stakeholders.
