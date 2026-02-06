# Gateway: Status Check

Check the current Gateway status and pending operations.

## Current Status

```bash
echo "=== Gateway Status ==="
echo ""

# Check for active workflows
echo "### Active Workflows"
if command -v gh &> /dev/null; then
  gh run list --limit 5 --json status,name,conclusion,createdAt 2>/dev/null || echo "Unable to fetch workflow runs"
else
  echo "Install GitHub CLI (gh) for workflow status"
fi

echo ""
echo "### Git Status"
git status -sb

echo ""
echo "### Recent Commits"
git log --oneline -5
```

## Pending Approvals

Check for pending workflow gates or approvals:

```bash
# Check for REVIEW_INSTRUCTIONS.md (pending implementation)
if [ -f REVIEW_INSTRUCTIONS.md ]; then
  echo "### Pending Review Implementation"
  echo "Found REVIEW_INSTRUCTIONS.md - implementation requested"
  head -20 REVIEW_INSTRUCTIONS.md
fi

# Check for workflow state
if [ -d .claude/artifacts/workflow-state ]; then
  echo "### Pending Workflow Gates"
  ls -la .claude/artifacts/workflow-state/
fi
```

## Platform Connection Status

| Platform | Status        | Last Activity                 |
| -------- | ------------- | ----------------------------- |
| GitHub   | Active        | Always via Actions            |
| Slack    | Check secrets | `SLACK_BOT_TOKEN` required    |
| Discord  | Check secrets | `DISCORD_BOT_TOKEN` required  |
| Telegram | Check secrets | `TELEGRAM_BOT_TOKEN` required |

## Quick Actions

From chat platforms, you can run:

- `/claude status` - This status check
- `/claude qa` - Run tests
- `/claude review` - Code review
- `/claude approve` - Approve pending gate

## GitHub Actions Commands

Common commands for the Gateway workflow:

```bash
# View recent workflow runs
gh run list --workflow=gateway-webhook.yml --limit 10

# Watch a running workflow
gh run watch

# Re-run a failed workflow
gh run rerun RUN_ID

# View workflow logs
gh run view RUN_ID --log
```

## Troubleshooting

### Commands Not Working

1. **Check workflow file exists**:

   ```bash
   ls -la .github/workflows/gateway-webhook.yml
   ```

2. **Check secrets are configured**:
   - Repository Settings > Secrets > Actions
   - Verify `ANTHROPIC_API_KEY` exists

3. **Check workflow permissions**:
   - Repository Settings > Actions > General
   - Enable "Allow GitHub Actions to create and approve pull requests"

### Response Not Posting

1. **Check notification config**:

   ```bash
   cat .claude/notifications.json 2>/dev/null || echo "No config found"
   ```

2. **Check platform tokens**:
   - Verify bot tokens haven't expired
   - Check OAuth scopes are sufficient

### Rate Limited

- Check platform-specific rate limits
- GitHub: 5000 requests/hour for authenticated
- Slack: Varies by tier
- Discord: 50 requests/second

---

**Gateway Status Check Complete**
