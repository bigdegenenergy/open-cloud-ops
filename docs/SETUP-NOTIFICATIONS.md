# Notification Setup Guide

This guide helps you configure failure notifications for both Claude Code (local) and GitHub Actions (CI/CD).

---

## ⚠️ MANDATORY: Read Source Repository First

Before configuring notifications, read **https://github.com/bigdegenenergy/ai-dev-toolkit** in full.

---

## Overview

The notification system supports multiple platforms:

| Platform       | Local Support | GitHub Actions Support |
| -------------- | ------------- | ---------------------- |
| Slack          | ✅            | ✅                     |
| Telegram       | ✅            | ✅                     |
| Discord        | ✅            | ✅                     |
| ntfy           | ✅            | ✅                     |
| Email          | ✅            | ✅                     |
| Custom Webhook | ✅            | ✅                     |

---

## Part 1: Local Claude Code Notifications

### Step 1: Create Configuration File

Copy the template and fill in your credentials:

```bash
cp .claude/notifications.json.template .claude/notifications.json
```

Edit `.claude/notifications.json` with your platform credentials.

### Step 2: Add to .gitignore

**IMPORTANT:** Never commit your credentials!

```bash
echo ".claude/notifications.json" >> .gitignore
```

### Step 3: Test Notifications

```bash
python3 .claude/hooks/notify.py --message "Test notification" --title "Test" --level info
```

### Alternative: Environment Variables

Instead of a config file, you can use environment variables:

```bash
# Slack
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/..."

# Telegram
export TELEGRAM_BOT_TOKEN="your-bot-token"
export TELEGRAM_CHAT_ID="your-chat-id"

# Discord
export DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/..."

# ntfy
export NTFY_SERVER="https://ntfy.sh"
export NTFY_TOPIC="your-topic"
export NTFY_TOKEN="optional-token"

# Email
export SMTP_HOST="smtp.gmail.com"
export SMTP_PORT="587"
export SMTP_USER="your-email@gmail.com"
export SMTP_PASSWORD="your-app-password"
export EMAIL_FROM="your-email@gmail.com"
export EMAIL_TO="recipient@example.com"

# Custom Webhook
export CUSTOM_WEBHOOK_URL="https://your-webhook.com/endpoint"
```

---

## Part 2: GitHub Actions Notifications

### Step 1: Add Repository Secrets

Go to your GitHub repository:

1. **Settings** → **Secrets and variables** → **Actions**
2. Click **New repository secret**
3. Add the relevant secrets for your platform(s)

### Required Secrets by Platform

#### GitHub Token (Required for Private Repos)

For private repositories, you MUST configure a `GH_TOKEN` with `repo` access for full workflow functionality:

| Secret Name | Description                             |
| ----------- | --------------------------------------- |
| `GH_TOKEN`  | Personal Access Token with `repo` scope |

**Why this is needed:**

- The default `GITHUB_TOKEN` has limited permissions in private repos
- Enables PR/issue comment automation (agent-reminder workflow)
- Required for cross-repository access

**How to get:**

1. Go to GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Generate new token with `repo` scope
3. Add as repository secret named `GH_TOKEN`

#### Slack

| Secret Name         | Description                |
| ------------------- | -------------------------- |
| `SLACK_WEBHOOK_URL` | Slack incoming webhook URL |

**How to get:** Slack App → Incoming Webhooks → Add New Webhook

#### Telegram

| Secret Name          | Description               |
| -------------------- | ------------------------- |
| `TELEGRAM_BOT_TOKEN` | Bot token from @BotFather |
| `TELEGRAM_CHAT_ID`   | Chat/Group/Channel ID     |

**How to get:**

1. Message @BotFather to create a bot
2. Get chat ID by messaging @userinfobot or from the API

#### Discord

| Secret Name           | Description         |
| --------------------- | ------------------- |
| `DISCORD_WEBHOOK_URL` | Discord webhook URL |

**How to get:** Server Settings → Integrations → Webhooks → New Webhook

#### ntfy

| Secret Name   | Description                       |
| ------------- | --------------------------------- |
| `NTFY_TOPIC`  | Your ntfy topic name              |
| `NTFY_SERVER` | (Optional) Self-hosted server URL |
| `NTFY_TOKEN`  | (Optional) Access token           |

**How to get:** Just pick a unique topic name at ntfy.sh

#### Email

| Secret Name     | Description                        |
| --------------- | ---------------------------------- |
| `SMTP_HOST`     | SMTP server (e.g., smtp.gmail.com) |
| `SMTP_PORT`     | SMTP port (usually 587)            |
| `SMTP_USER`     | SMTP username                      |
| `SMTP_PASSWORD` | SMTP password or app password      |
| `EMAIL_FROM`    | Sender email address               |
| `EMAIL_TO`      | Recipient email address            |

**How to get (Gmail):**

1. Enable 2FA on your Google account
2. Generate an App Password
3. Use the app password as SMTP_PASSWORD

#### Custom Webhook

| Secret Name          | Description           |
| -------------------- | --------------------- |
| `CUSTOM_WEBHOOK_URL` | Your webhook endpoint |

---

## Part 3: Platform-Specific Setup Guides

### Slack Setup

1. Go to [Slack API](https://api.slack.com/apps)
2. Create New App → From scratch
3. Enable **Incoming Webhooks**
4. Add New Webhook to Workspace
5. Select the channel
6. Copy the webhook URL

### Telegram Setup

1. Message [@BotFather](https://t.me/BotFather) on Telegram
2. Send `/newbot` and follow instructions
3. Copy the bot token
4. Start a chat with your bot
5. Get your chat ID:
   ```bash
   curl https://api.telegram.org/bot<YOUR_TOKEN>/getUpdates
   ```
   Look for `"chat":{"id":...}`

### Discord Setup

1. Open Discord Server Settings
2. Go to **Integrations** → **Webhooks**
3. Click **New Webhook**
4. Name it and select a channel
5. Copy the Webhook URL

### ntfy Setup

1. Go to [ntfy.sh](https://ntfy.sh)
2. Choose a unique topic name
3. Subscribe on your devices
4. (Optional) Set up access tokens for private topics

### Email Setup (Gmail)

1. Enable 2-Factor Authentication
2. Go to [Google App Passwords](https://myaccount.google.com/apppasswords)
3. Generate a new App Password for "Mail"
4. Use this password as `SMTP_PASSWORD`

---

## Part 4: Testing

### Test Local Notifications

```bash
# Test all configured platforms
python3 .claude/hooks/notify.py -m "Test message" -t "Test" -l info

# Test specific platform
python3 .claude/hooks/notify.py -m "Test" -t "Test" -l info -p slack
python3 .claude/hooks/notify.py -m "Test" -t "Test" -l error -p telegram
```

### Test GitHub Actions

1. Create a test branch
2. Intentionally break a test or linting rule
3. Push and verify notification arrives

---

## Part 5: Integration with Stop Hook

To send notifications when Claude Code tasks fail, update your stop hook:

```bash
# In .claude/hooks/stop.sh, add at the end:
if [ $EXIT_CODE -ne 0 ]; then
    python3 "$(dirname "$0")/notify.py" \
        --title "Claude Code Task Failed" \
        --message "Task failed with exit code $EXIT_CODE" \
        --level error
fi
```

---

## Troubleshooting

### Notifications not sending

1. Check credentials are correct
2. Verify config file exists and is valid JSON
3. Check network connectivity
4. Review error output from notify.py

### GitHub Actions not triggering

1. Verify secrets are set correctly (no trailing spaces)
2. Check workflow file syntax
3. Ensure the notification workflow is enabled

### Rate limiting

- **Slack:** 1 message per second
- **Telegram:** 30 messages per second to same chat
- **Discord:** 30 requests per minute
- **ntfy:** Varies by server

---

## Security Best Practices

1. **Never commit credentials** - Use .gitignore
2. **Use app passwords** - Never use main account passwords
3. **Rotate tokens** - Periodically regenerate webhook URLs and tokens
4. **Limit permissions** - Use least privilege for bot permissions
5. **Review access** - Audit who has access to notification channels

---

## Quick Reference

### Minimum Setup (Choose One)

**Slack (Easiest):**

```bash
# Local
echo '{"slack":{"webhook_url":"YOUR_URL"}}' > .claude/notifications.json

# GitHub
# Add SLACK_WEBHOOK_URL secret
```

**ntfy (No Account Required):**

```bash
# Local
echo '{"ntfy":{"topic":"my-unique-topic"}}' > .claude/notifications.json

# GitHub
# Add NTFY_TOPIC secret
```

**Telegram (Free):**

```bash
# Local
echo '{"telegram":{"bot_token":"TOKEN","chat_id":"ID"}}' > .claude/notifications.json

# GitHub
# Add TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID secrets
```
