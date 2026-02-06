# Anthropic Claude Models Reference

This document provides the official model IDs for Claude models used in this repository.

## Claude 4.5 Family (Current)

| Model                 | Model ID                     | Release Date   | Use Case                                               |
| --------------------- | ---------------------------- | -------------- | ------------------------------------------------------ |
| **Claude 4.5 Opus**   | `claude-opus-4-5-20251101`   | November 2025  | Complex reasoning, architecture, code review, planning |
| **Claude 4.5 Sonnet** | `claude-sonnet-4-5-20250929` | September 2025 | Balanced performance, general coding tasks             |
| **Claude 4.5 Haiku**  | `claude-haiku-4-5-20251001`  | October 2025   | Fast tasks, simple parsing, comment analysis           |

## Model Selection Guidelines

### When to Use Opus (`claude-opus-4-5-20251101`)

- Complex architectural decisions
- Security audits and code reviews
- Multi-step planning and reasoning
- Tasks requiring deep understanding
- Production-critical code generation

**Example workflows:**

- `/plan` - Planning complex features
- `/review` - Critical code review
- `@security-auditor` - Security analysis
- GitHub Actions for PR implementation

### When to Use Sonnet (`claude-sonnet-4-5-20250929`)

- General coding tasks
- Documentation generation
- Test writing
- Routine refactoring
- Balanced cost/performance needs

**Example workflows:**

- `@verify-app` - Application verification
- Standard code generation
- Documentation updates

### When to Use Haiku (`claude-haiku-4-5-20251001`)

- Simple, fast tasks
- Comment/intent parsing
- Quick validations
- High-volume, low-complexity operations
- Cost-sensitive workflows

**Example workflows:**

- Analyzing PR comments for intent
- Simple code transformations
- Implementing pre-approved plans

## Configuration Examples

### In Slash Commands (`.claude/commands/*.md`)

```yaml
---
name: my-command
description: Description of the command
model: claude-opus-4-5-20251101
---
```

### In Subagents (`.claude/agents/*.md`)

```yaml
---
name: my-agent
description: Description of the agent
tools: Read, Edit, Grep, Glob
model: claude-opus-4-5-20251101
---
```

### In GitHub Actions

```yaml
- uses: anthropics/claude-code-action@v1
  with:
    anthropic_api_key: ${{ secrets.ANTHROPIC_API_KEY }}
    claude_args: |
      --model claude-opus-4-5-20251101
```

### In API Calls

```bash
curl -s "https://api.anthropic.com/v1/messages" \
  -H "Content-Type: application/json" \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-haiku-4-5-20251001",
    "max_tokens": 500,
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

### In Settings (`.claude/settings.json`)

```json
{
  "model": "claude-opus-4-5-20251101"
}
```

### In Bootstrap Config (`.claude/bootstrap.toml`)

```toml
[agent.identity]
model_preference = "claude-opus-4-5-20251101"

[best_practices.model_selection]
default = "claude-opus-4-5-20251101"
```

## Model ID Format

Anthropic model IDs follow the format:

```
claude-{tier}-{version}-{YYYYMMDD}
```

- **tier**: `opus`, `sonnet`, or `haiku`
- **version**: `4-5` (for Claude 4.5 family)
- **YYYYMMDD**: Release date

## Important Notes

1. **Always use full model IDs** - Short forms like `opus`, `sonnet`, or `haiku` may not work in all contexts
2. **Check for updates** - Model IDs can change with new releases
3. **API compatibility** - Ensure your `anthropic-version` header is compatible with the model

## References

- [Anthropic Models Documentation](https://docs.anthropic.com/en/docs/about-claude/models)
- [Claude Code CLI Documentation](https://docs.anthropic.com/en/docs/claude-code)
