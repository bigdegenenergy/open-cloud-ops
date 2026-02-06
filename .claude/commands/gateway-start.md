# Gateway: Start ChatOps Connection

Start the Gateway for remote control via chat platforms.

## Overview

This command sets up the Gateway for receiving commands from Slack, Discord, or Telegram.

## Configuration Status

Check your platform configuration:

```bash
# Check for notification config (used for responses)
if [ -f .claude/notifications.json ]; then
  echo "Notification config found:"
  cat .claude/notifications.json | grep -E '"slack"|"discord"|"telegram"' | head -5
else
  echo "No notification config found. Copy from template:"
  echo "cp .claude/notifications.json.template .claude/notifications.json"
fi
```

## GitHub Actions Integration

The Gateway uses GitHub Actions `repository_dispatch` to receive commands. Ensure the workflow is enabled:

```yaml
# .github/workflows/gateway-webhook.yml
name: Gateway Webhook

on:
  repository_dispatch:
    types: [chat-command]

jobs:
  execute:
    runs-on: ubuntu-latest
    steps:
      - uses: anthropics/claude-code-action@v1
        with:
          anthropic_api_key: ${{ secrets.ANTHROPIC_API_KEY }}
          prompt: ${{ github.event.client_payload.command }}
```

## Platform Setup

### Slack

1. Create a Slack App at https://api.slack.com/apps
2. Add Slash Command: `/claude`
3. Set Request URL: `https://api.github.com/repos/OWNER/REPO/dispatches`
4. Add scopes: `chat:write`, `commands`
5. Store bot token in GitHub Secrets as `SLACK_BOT_TOKEN`

### Discord

1. Create Discord Application: https://discord.com/developers
2. Add bot to your server
3. Configure bot to call GitHub repository_dispatch
4. Store token in GitHub Secrets as `DISCORD_BOT_TOKEN`

### Telegram

1. Create bot via @BotFather on Telegram
2. Set webhook URL to your endpoint
3. Store token in GitHub Secrets as `TELEGRAM_BOT_TOKEN`

## Available Commands via Chat

Once configured, users can send these commands:

| Command               | Description                 | Permission |
| --------------------- | --------------------------- | ---------- |
| `/claude plan <desc>` | Plan feature implementation | member     |
| `/claude qa`          | Run tests and fix           | member     |
| `/claude review`      | Critical code review        | member     |
| `/claude zeno <path>` | Surgical code analysis      | member     |
| `/claude ship`        | Commit and create PR        | admin      |
| `/claude status`      | Check session status        | viewer     |
| `/claude approve`     | Approve workflow gate       | admin      |
| `/claude help`        | Show available commands     | viewer     |

## User Mapping

To map chat users to GitHub permissions, create `.claude/gateway/users.json`:

```json
{
  "slack": {
    "U12345678": { "github": "username", "permission": "admin" },
    "U87654321": { "github": "other-user", "permission": "member" }
  },
  "discord": {
    "123456789012345678": { "github": "username", "permission": "admin" }
  },
  "telegram": {
    "12345678": { "github": "username", "permission": "admin" }
  }
}
```

## Testing the Connection

After setup, test with:

```bash
# Trigger a test dispatch
curl -X POST https://api.github.com/repos/OWNER/REPO/dispatches \
  -H "Authorization: token YOUR_PAT" \
  -H "Accept: application/vnd.github+json" \
  -d '{"event_type": "chat-command", "client_payload": {"command": "/claude help", "platform": "test"}}'
```

## Verification Checklist

- [ ] GitHub Actions workflow enabled
- [ ] `ANTHROPIC_API_KEY` secret configured
- [ ] Platform-specific secrets configured
- [ ] User mapping file created (optional)
- [ ] Test command successful

## Security Notes

1. **Secrets**: Never expose API tokens in logs
2. **Permissions**: Use minimal required permissions
3. **Rate Limiting**: Platforms enforce their own rate limits
4. **Audit**: All commands are logged in GitHub Actions

---

**Gateway configured and ready for chat commands.**
