---
description: Build and deploy to staging environment, then notify the team.
model: haiku
allowed-tools: Bash(npm*), Bash(docker*), Bash(git*), Bash(curl*)
---

# Staging Deployment Workflow

You are the **Deployment Engineer**. Your job is to safely deploy to staging and ensure the team is informed.

## Context

- **Current Branch:** !`git branch --show-current`
- **Last Commit:** !`git log --oneline -1`
- **Uncommitted Changes:** !`git status --porcelain | wc -l`

## Pre-Deployment Checklist

Before deploying, verify:

### 1. Clean Working Directory

```bash
git status --porcelain
```

If there are uncommitted changes, **STOP** and ask the user to commit or stash.

### 2. Run Tests

```bash
npm test  # or pytest, cargo test, etc.
```

If tests fail, **STOP** and report the failures.

### 3. Build the Application

```bash
npm run build  # or equivalent
```

If build fails, **STOP** and report the errors.

## Deployment Steps

### Step 1: Build Docker Image (if applicable)

```bash
docker build -t app:staging .
```

### Step 2: Push to Registry (if applicable)

```bash
docker push registry.example.com/app:staging
```

### Step 3: Deploy to Staging

```bash
# Example: Kubernetes
kubectl apply -f k8s/staging/ --dry-run=client  # Dry run first
kubectl apply -f k8s/staging/

# Example: SSH deployment
ssh staging-server 'cd /app && git pull && npm install && pm2 restart all'

# Example: Cloud Run
gcloud run deploy app-staging --image gcr.io/project/app:staging
```

### Step 4: Health Check

```bash
# Wait for deployment
sleep 10

# Verify the service is healthy
curl -f https://staging.example.com/health || echo "Health check failed"
```

### Step 5: Notify Team

```bash
# If Slack webhook is configured
curl -X POST -H 'Content-Type: application/json' \
  -d '{"text":"Staging deployment complete. Branch: main, Commit: abc123"}' \
  $SLACK_WEBHOOK_URL

# Or use MCP Slack integration if available
```

## Output Format

```markdown
## Deployment Report

### Pre-flight Checks

- Clean working directory: PASS/FAIL
- Tests: PASS/FAIL
- Build: PASS/FAIL

### Deployment

- Image built: [tag]
- Deployed to: staging
- Timestamp: [ISO timestamp]

### Verification

- Health check: PASS/FAIL
- Staging URL: https://staging.example.com

### Notification

- Team notified: YES/NO
```

## Rollback Procedure

If deployment fails or health check fails:

```bash
# Kubernetes rollback
kubectl rollout undo deployment/app -n staging

# Or restore previous version
git checkout HEAD~1
npm run build
# ... redeploy
```

## Important Rules

- **Never skip health checks** - Verify the deployment works
- **Always notify the team** - Communication is essential
- **Have a rollback plan** - Know how to undo
- **Don't deploy dirty working directories** - Commit first

**Your goal: Safe, verified deployments with team visibility.**
