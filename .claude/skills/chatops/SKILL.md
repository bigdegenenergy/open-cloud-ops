# ChatOps Skill

> Remote control of development workflows via chat platforms (Slack, Discord, Telegram).

## Overview

ChatOps enables bidirectional communication between developers and Claude Code through chat platforms. Commands can be triggered from chat, and results are posted back to the conversation.

## Supported Platforms

| Platform | Command Prefix | Bot Type      |
| -------- | -------------- | ------------- |
| Slack    | `/claude`      | Slash command |
| Discord  | `!claude`      | Bot command   |
| Telegram | `/claude`      | Bot command   |
| GitHub   | `@claude`      | Comment       |

## Command Mapping

Chat commands map to Claude Code commands:

| Chat Command          | Claude Command      | Description                   |
| --------------------- | ------------------- | ----------------------------- |
| `/claude plan <desc>` | `/plan`             | Plan a feature                |
| `/claude qa`          | `/qa`               | Run tests and fix             |
| `/claude review`      | `/review`           | Critical code review          |
| `/claude zeno <path>` | `/zeno`             | Surgical code analysis        |
| `/claude ship`        | `/ship`             | Commit and create PR          |
| `/claude status`      | `/gateway-status`   | Check current session status  |
| `/claude approve`     | `/workflow-approve` | Approve pending workflow gate |

## Architecture

```
Chat Platform
     |
     v
[Webhook Receiver] ──────> GitHub Actions (workflow_dispatch)
     |                              |
     v                              v
[Command Parser] <──────── Claude Code Execution
     |                              |
     v                              v
[Response Formatter] ──────> Chat Response
```

## Authentication Flow

```
1. User sends command in chat
2. Platform identifies user (Slack ID, Discord ID, etc.)
3. Gateway maps chat user to GitHub username
4. Verifies user has permission for command
5. Executes command with user context
6. Returns result to chat thread
```

## Permission Levels

| Level  | Commands Allowed                             |
| ------ | -------------------------------------------- |
| viewer | status, help                                 |
| member | qa, review, zeno, status                     |
| admin  | All commands including ship, approve, deploy |

## Response Formatting

### Slack

```json
{
  "blocks": [
    {
      "type": "section",
      "text": {
        "type": "mrkdwn",
        "text": "*QA Results*\n:white_check_mark: 42 tests passed\n:x: 0 tests failed"
      }
    }
  ]
}
```

### Discord

```markdown
**QA Results**
:white_check_mark: 42 tests passed
:x: 0 tests failed
```

### Telegram

```
QA Results

42 tests passed
0 tests failed
```

## Integration Patterns

### Pattern 1: Trigger from Chat

```
User: /claude qa
Bot: Running QA checks...
Bot: [5 min later] QA Complete: 42 tests passed, 3 fixed
```

### Pattern 2: Approval via Chat

```
Bot: [Workflow Paused] Ready to deploy to staging?
     React with :thumbsup: to approve or reply "reject"
User: :thumbsup:
Bot: Deploying to staging...
Bot: Deployment complete! https://staging.example.com
```

### Pattern 3: Status Updates

```
Bot: [CI Update] PR #123: Tests passing, review requested
User: /claude review 123
Bot: Reviewing PR #123...
Bot: Review complete: 2 suggestions posted
```

## Implementation via GitHub Actions

Leverage existing `@claude` workflow for chat commands:

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
          prompt: ${{ github.event.client_payload.command }}

      - name: Post Response
        run: |
          curl -X POST "${{ github.event.client_payload.callback_url }}" \
            -H "Content-Type: application/json" \
            -d '{"text": "${{ steps.claude.outputs.response }}"}'
```

## Setting Up Chat Integrations

### Slack

1. Create Slack App at https://api.slack.com/apps
2. Add Slash Command: `/claude`
3. Set Request URL to your webhook endpoint
4. Add Bot Token Scopes: `chat:write`, `commands`
5. Store `SLACK_BOT_TOKEN` in GitHub Secrets

### Discord

1. Create Discord Application at https://discord.com/developers
2. Create Bot and get token
3. Add bot to server with message permissions
4. Store `DISCORD_BOT_TOKEN` in GitHub Secrets

### Telegram

1. Create bot via @BotFather
2. Get bot token
3. Set webhook URL
4. Store `TELEGRAM_BOT_TOKEN` in GitHub Secrets

## Error Handling

```
User: /claude deploy production
Bot: Error: You don't have permission to deploy to production.
     Required role: admin
     Your role: member

     Contact @admin-user to request elevated permissions.
```

## Rate Limiting

| Scope       | Limit           |
| ----------- | --------------- |
| Per user    | 10 commands/min |
| Per channel | 30 commands/min |
| Per org     | 100 commands/hr |

## Activation Triggers

This skill activates when prompts contain:

- "chatops", "slack bot", "discord bot", "telegram bot"
- "remote command", "chat trigger", "webhook"
- "gateway", "bidirectional", "chat platform"

## Security Considerations

1. **Token Security**: Never log tokens, use secrets
2. **Input Validation**: Sanitize all chat inputs
3. **Rate Limiting**: Prevent abuse
4. **Audit Logging**: Track all commands executed
5. **Timeout Handling**: Cancel long-running commands
