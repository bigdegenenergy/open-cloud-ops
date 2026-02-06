# Claude Code Configuration Directory

This directory contains the complete professional engineering team setup for Claude Code.

## Quick Reference

### For AI Agents

If you're an AI agent working in this repository, start by reading **`bootstrap.toml`** - it contains all the initialization instructions, workflows, and best practices you need to follow.

```bash
cat .claude/bootstrap.toml
```

### For Human Developers

This directory contains:

- **`bootstrap.toml`** - AI agent initialization and workflow configuration
- **`commands/`** - Slash commands for common workflows
- **`agents/`** - Specialized subagents (team members)
- **`hooks/`** - Automated quality gates
- **`settings.json`** - Team-wide configurations
- **`docs.md`** - Living team knowledge base

## Directory Structure

```
.claude/
├── bootstrap.toml          # AI agent bootstrap configuration
├── commands/               # Slash commands
│   ├── git/
│   │   └── commit-push-pr.md
│   ├── test/
│   └── deploy/
├── agents/                 # Subagents (specialized team members)
│   ├── code-simplifier.md
│   ├── verify-app.md
│   └── code-reviewer.md
├── hooks/                  # Automated quality gates
│   ├── post-tool-use.sh
│   └── stop.sh
├── settings.json           # Team-wide configurations
├── docs.md                 # Team knowledge base
└── README.md               # This file
```

## Usage

### For AI Agents

1. **Read bootstrap.toml first** - Contains all initialization instructions
2. **Follow the standard workflow** - Plan mode → Implementation → Subagents → Commit
3. **Use subagents proactively** - @code-simplifier, @verify-app, @code-reviewer
4. **Always verify your work** - Quality gates are mandatory
5. **Update docs.md weekly** - Add new patterns and learnings

### For Developers

1. **Customize docs.md** - Add project-specific conventions
2. **Adjust settings.json** - Configure permissions for your environment
3. **Create new commands** - Add slash commands for repeated workflows
4. **Refine subagents** - Improve prompts based on experience
5. **Track metrics** - Monitor usage in `.claude/metrics/`

## Key Files

### bootstrap.toml
Complete AI agent configuration including:
- Identity and role definition
- Standard workflows
- Subagent specifications
- Hook configurations
- Best practices and anti-patterns
- Quality gates
- Onboarding instructions

### commands/
Reusable slash commands for inner loop workflows:
- Pre-compute context with inline bash
- Security controls via allowed-tools
- Version controlled and team-shared

### agents/
Specialized AI team members:
- **code-simplifier** - Code hygiene after changes
- **verify-app** - QA testing before commits
- **code-reviewer** - Critical review before PRs

### hooks/
Automated quality gates:
- **post-tool-use.sh** - Format code after edits
- **stop.sh** - Run tests at end of turn

### settings.json
Team-wide configurations:
- Pre-approved safe commands
- Hook enablement
- Default model settings

### docs.md
Living team knowledge base:
- Project conventions
- Common patterns
- Anti-patterns to avoid
- Known issues and workarounds

## Getting Started

### For AI Agents
```bash
# Read the bootstrap configuration
cat .claude/bootstrap.toml

# Start working with the standard workflow
# 1. Plan mode (shift+tab twice)
# 2. Implementation
# 3. Quality checks with subagents
# 4. Commit with /commit-push-pr
```

### For Developers
```bash
# Customize for your project
vim .claude/docs.md
vim .claude/settings.json

# Make hooks executable
chmod +x .claude/hooks/*.sh

# Start using Claude Code
claude-code
```

## Philosophy

This setup is based on Boris Cherny's (creator of Claude Code) actual workflow and follows one key principle:

> "Give Claude a way to verify its work. If Claude has that feedback loop, it will 2-3x the quality of the final result."

Every workflow includes verification. Every change goes through quality gates. Every pattern is documented for the team.

## Resources

- **Main README** - `../README.md`
- **Research Report** - `../RESEARCH.md`
- **Implementation Guide** - `../IMPLEMENTATION_GUIDE.md`
- **Official Docs** - https://code.claude.com/docs/
- **Boris's Thread** - https://x.com/bcherny/status/2007179847949500714

---

**For AI Agents:** Start with `bootstrap.toml` - it contains everything you need to work professionally in this repository.

**For Developers:** Customize this setup for your team and commit it to Git for consistency across all team members.
