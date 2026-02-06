---
description: Production incident response workflow. Guides through assessment, mitigation, and resolution.
---

# Incident Response Workflow

You are guiding the user through a production incident response following structured incident management practices.

## Incident Severity Levels

| Level | Description | Response Time | Escalation |
|-------|-------------|---------------|------------|
| SEV-1 | Complete outage | Immediate | All hands |
| SEV-2 | Major degradation | 15 minutes | On-call team |
| SEV-3 | Minor impact | 1 hour | Primary on-call |
| SEV-4 | No user impact | Next business day | During work hours |

## Phase 1: Assessment (First 5 minutes)

### Quick Questions
1. **What is the impact?** (users affected, services down)
2. **When did it start?** (check monitoring, user reports)
3. **What changed?** (deployments, config changes, traffic spikes)
4. **Is it getting worse?**

### Immediate Actions
```bash
# Check service health
kubectl get pods -A | grep -v Running
kubectl get events --sort-by='.lastTimestamp' | tail -20

# Check recent deployments
kubectl rollout history deployment/<name>

# Check resource usage
kubectl top pods
kubectl top nodes
```

## Phase 2: Mitigation (Stop the bleeding)

### Rollback Decision Tree
```
Is this caused by a recent deployment?
├── YES → Rollback immediately
│   kubectl rollout undo deployment/<name>
└── NO → Continue investigation
    ├── Can we scale up? → kubectl scale deployment/<name> --replicas=X
    ├── Can we redirect traffic? → Update ingress/load balancer
    └── Can we disable the feature? → Feature flag toggle
```

### Communication
- [ ] Acknowledge incident in Slack/PagerDuty
- [ ] Post status update to status page
- [ ] Notify stakeholders of impact and ETA

## Phase 3: Investigation

Use specialized agents for debugging:

```
# For infrastructure issues
Invoke @devops-troubleshooter to analyze infrastructure

# For Kubernetes issues
Invoke @kubernetes-architect for K8s debugging

# For application issues
Invoke @python-pro or @typescript-pro for code analysis
```

### Log Analysis
```bash
# Find error patterns
kubectl logs <pod> --since=1h | grep -i error | sort | uniq -c

# Check specific timeframe
kubectl logs <pod> --since-time="2024-01-15T10:00:00Z"

# Follow live logs
kubectl logs <pod> -f | grep -E "(ERROR|WARN)"
```

### Common Issues Checklist
- [ ] OOMKilled pods (memory limits)
- [ ] CrashLoopBackOff (application errors)
- [ ] Connection refused (service discovery)
- [ ] Database connection pool exhaustion
- [ ] External API failures
- [ ] Certificate expiration
- [ ] DNS resolution failures

## Phase 4: Resolution

### Verify the Fix
```bash
# Check pod status
kubectl get pods -o wide

# Check endpoints
kubectl get endpoints <service>

# Test connectivity
kubectl exec -it <pod> -- curl <service>:port/health
```

### Post-Incident
- [ ] Verify all metrics back to normal
- [ ] Update status page: Resolved
- [ ] Schedule postmortem meeting
- [ ] Create follow-up tickets

## Phase 5: Postmortem

### Template
```markdown
# Incident Postmortem: [Title]

## Summary
- **Date**: YYYY-MM-DD
- **Duration**: X hours Y minutes
- **Severity**: SEV-X
- **Impact**: [User-facing impact]

## Timeline
- HH:MM - Incident detected
- HH:MM - Mitigation applied
- HH:MM - Root cause identified
- HH:MM - Fix deployed
- HH:MM - Incident resolved

## Root Cause
[Detailed technical explanation]

## What Went Well
-

## What Went Poorly
-

## Action Items
- [ ] [Action 1] - Owner - Due Date
- [ ] [Action 2] - Owner - Due Date

## Lessons Learned
-
```

## Current Cluster Status

**Pods:** !`kubectl get pods -A --no-headers 2>/dev/null | grep -v Running | head -10 || echo "Unable to check cluster status"`

**Recent Events:** !`kubectl get events --sort-by='.lastTimestamp' -A 2>/dev/null | tail -5 || echo "Unable to fetch events"`

**Important**: Document everything during the incident. Decisions, actions, and timestamps are valuable for the postmortem.
