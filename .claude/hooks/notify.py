#!/usr/bin/env python3
"""
Multi-platform notification system for Claude Code failures.

Supports: Slack, Telegram, Email, ntfy, Discord, and custom webhooks.

Usage:
    python3 notify.py --message "Build failed" --title "CI Error"
    python3 notify.py --message "Tests failed" --level error
    python3 notify.py --message "Deploy complete" --level success

Configuration:
    Set credentials in ~/.claude/notifications.json or .claude/notifications.json
"""

import argparse
import json
import os
import sys
import urllib.request
import urllib.error
import smtplib
from email.mime.text import MIMEText
from email.mime.multipart import MIMEMultipart
from pathlib import Path


def load_config():
    """Load notification configuration from multiple possible locations."""
    config_paths = [
        Path.home() / ".claude" / "notifications.json",
        Path(".claude") / "notifications.json",
        Path("notifications.json"),
    ]

    for path in config_paths:
        if path.exists():
            try:
                with open(path) as f:
                    return json.load(f)
            except json.JSONDecodeError:
                print(f"Warning: Invalid JSON in {path}", file=sys.stderr)

    # Fall back to environment variables
    return {
        "slack": {"webhook_url": os.environ.get("SLACK_WEBHOOK_URL")},
        "telegram": {
            "bot_token": os.environ.get("TELEGRAM_BOT_TOKEN"),
            "chat_id": os.environ.get("TELEGRAM_CHAT_ID"),
        },
        "discord": {"webhook_url": os.environ.get("DISCORD_WEBHOOK_URL")},
        "ntfy": {
            "server": os.environ.get("NTFY_SERVER", "https://ntfy.sh"),
            "topic": os.environ.get("NTFY_TOPIC"),
            "token": os.environ.get("NTFY_TOKEN"),
        },
        "email": {
            "smtp_host": os.environ.get("SMTP_HOST"),
            "smtp_port": int(os.environ.get("SMTP_PORT", "587")),
            "smtp_user": os.environ.get("SMTP_USER"),
            "smtp_password": os.environ.get("SMTP_PASSWORD"),
            "from_address": os.environ.get("EMAIL_FROM"),
            "to_address": os.environ.get("EMAIL_TO"),
        },
        "webhook": {"url": os.environ.get("CUSTOM_WEBHOOK_URL")},
    }


def send_slack(config, title, message, level):
    """Send notification to Slack via webhook."""
    webhook_url = config.get("webhook_url")
    if not webhook_url:
        return False

    color = {"error": "#FF0000", "warning": "#FFA500", "success": "#00FF00", "info": "#0000FF"}.get(level, "#808080")

    payload = {
        "attachments": [{
            "color": color,
            "title": title,
            "text": message,
            "footer": "Claude Code Notifications",
        }]
    }

    return _send_json_request(webhook_url, payload)


def send_telegram(config, title, message, level):
    """Send notification to Telegram."""
    bot_token = config.get("bot_token")
    chat_id = config.get("chat_id")

    if not bot_token or not chat_id:
        return False

    emoji = {"error": "🔴", "warning": "🟡", "success": "🟢", "info": "🔵"}.get(level, "⚪")

    text = f"{emoji} *{title}*\n\n{message}"

    url = f"https://api.telegram.org/bot{bot_token}/sendMessage"
    payload = {
        "chat_id": chat_id,
        "text": text,
        "parse_mode": "Markdown",
    }

    return _send_json_request(url, payload)


def send_discord(config, title, message, level):
    """Send notification to Discord via webhook."""
    webhook_url = config.get("webhook_url")
    if not webhook_url:
        return False

    color = {"error": 0xFF0000, "warning": 0xFFA500, "success": 0x00FF00, "info": 0x0000FF}.get(level, 0x808080)

    payload = {
        "embeds": [{
            "title": title,
            "description": message,
            "color": color,
            "footer": {"text": "Claude Code Notifications"},
        }]
    }

    return _send_json_request(webhook_url, payload)


def send_ntfy(config, title, message, level):
    """Send notification to ntfy.sh or self-hosted ntfy server."""
    server = config.get("server", "https://ntfy.sh")
    topic = config.get("topic")
    token = config.get("token")

    if not topic:
        return False

    url = f"{server.rstrip('/')}/{topic}"

    priority = {"error": "5", "warning": "4", "success": "3", "info": "3"}.get(level, "3")
    tags = {"error": "x", "warning": "warning", "success": "white_check_mark", "info": "information_source"}.get(level, "")

    headers = {
        "Title": title,
        "Priority": priority,
        "Tags": tags,
    }

    if token:
        headers["Authorization"] = f"Bearer {token}"

    try:
        req = urllib.request.Request(url, data=message.encode(), headers=headers, method="POST")
        with urllib.request.urlopen(req, timeout=10) as response:
            return response.status == 200
    except Exception as e:
        print(f"ntfy error: {e}", file=sys.stderr)
        return False


def send_email(config, title, message, level):
    """Send notification via email."""
    required = ["smtp_host", "smtp_user", "smtp_password", "from_address", "to_address"]
    if not all(config.get(k) for k in required):
        return False

    try:
        msg = MIMEMultipart()
        msg["From"] = config["from_address"]
        msg["To"] = config["to_address"]
        msg["Subject"] = f"[{level.upper()}] {title}"

        body = f"{title}\n\n{message}\n\n--\nClaude Code Notifications"
        msg.attach(MIMEText(body, "plain"))

        with smtplib.SMTP(config["smtp_host"], config.get("smtp_port", 587)) as server:
            server.starttls()
            server.login(config["smtp_user"], config["smtp_password"])
            server.send_message(msg)

        return True
    except Exception as e:
        print(f"Email error: {e}", file=sys.stderr)
        return False


def send_webhook(config, title, message, level):
    """Send notification to a custom webhook."""
    url = config.get("url")
    if not url:
        return False

    payload = {
        "title": title,
        "message": message,
        "level": level,
        "source": "claude-code",
    }

    return _send_json_request(url, payload)


def _send_json_request(url, payload):
    """Send a JSON POST request."""
    try:
        data = json.dumps(payload).encode()
        req = urllib.request.Request(
            url,
            data=data,
            headers={"Content-Type": "application/json"},
            method="POST"
        )
        with urllib.request.urlopen(req, timeout=10) as response:
            return response.status in [200, 201, 204]
    except Exception as e:
        print(f"Request error: {e}", file=sys.stderr)
        return False


def main():
    parser = argparse.ArgumentParser(description="Send notifications to multiple platforms")
    parser.add_argument("--message", "-m", required=True, help="Notification message")
    parser.add_argument("--title", "-t", default="Claude Code Alert", help="Notification title")
    parser.add_argument("--level", "-l", choices=["error", "warning", "success", "info"], default="info", help="Alert level")
    parser.add_argument("--platform", "-p", action="append", help="Specific platform(s) to notify (default: all configured)")
    args = parser.parse_args()

    config = load_config()

    senders = {
        "slack": send_slack,
        "telegram": send_telegram,
        "discord": send_discord,
        "ntfy": send_ntfy,
        "email": send_email,
        "webhook": send_webhook,
    }

    platforms = args.platform if args.platform else senders.keys()

    results = {}
    for platform in platforms:
        if platform in senders and platform in config:
            results[platform] = senders[platform](config[platform], args.title, args.message, args.level)

    # Print results
    sent_to = [p for p, success in results.items() if success]
    failed = [p for p, success in results.items() if not success and config.get(p)]

    if sent_to:
        print(f"Notification sent to: {', '.join(sent_to)}")
    if failed:
        print(f"Failed to send to: {', '.join(failed)}", file=sys.stderr)

    if not sent_to and not failed:
        print("No notification platforms configured", file=sys.stderr)
        sys.exit(1)

    sys.exit(0 if sent_to else 1)


if __name__ == "__main__":
    main()
