# Research Synthesis: Gap Fill Implementation

> **7 Research Agents Completed** | **Synthesized: 2026-01-04**

This document synthesizes findings from 7 parallel research agents investigating Claude Code best practices. Each pillar now has confirmed specifications, code examples, and implementation-ready guidance.

---

## Executive Summary

| Pillar | Key Findings | Implementation Complexity |
|--------|--------------|---------------------------|
| 1. Extended Thinking | 4 trigger levels, CLI-only, case-insensitive | Low |
| 2. Session Management | 10+ commands, checkpoint system, 60% myth debunked | Low |
| 3. MCP Servers | Official + 7,500+ community servers, `.mcp.json` format | Medium |
| 4. Skills | Auto-discovery via description, `.claude/skills/` | Medium |
| 5. Modular CLAUDE.md | `.claude/rules/` official (v2.0.64+), path-scoped rules | Low |
| 6. Visual & Headless | Puppeteer deprecated → Playwright, `-p` flag patterns | Medium |
| 7. Multi-Claude | Worktrees + subagents, 10 parallel cap, context isolation | High |

---

## Pillar 1: Extended Thinking - CONFIRMED

### Trigger Keywords (Claude Code CLI Only)

| Trigger | Budget | Use Case |
|---------|--------|----------|
| `think` | ~4,000 tokens | Routine fixes, refactoring |
| `think hard` / `megathink` | ~10,000 tokens | Multi-step algorithms, caching |
| `think harder` / `ultrathink` | ~31,999 tokens | Architecture, security audits |

**Key Findings:**
- **Case-insensitive**: Implementation uses `toLowerCase()` preprocessing
- **CLI-only**: Does NOT work in claude.ai web or direct API
- **Position-flexible**: Can appear anywhere in prompt
- **Cost**: Thinking tokens billed as output tokens (same rate)

### Verbose Mode
- **Toggle**: `Ctrl+O`
- **Display**: Gray italic text (in supported terminals)
- **v2.0.0 regression**: Must use Ctrl+O → Ctrl+E → scroll to view thinking

### Implementation
```markdown
## Thinking Triggers (Claude Code CLI Only)

| Trigger | Budget | When to Use |
|---------|--------|-------------|
| "think" | ~4k tokens | Simple fixes, refactoring |
| "think hard" | ~10k tokens | Complex algorithms, API design |
| "ultrathink" | ~32k tokens | Major architecture, security audits |

**Anti-pattern**: Don't use ultrathink for everything (3-8x cost)
**Verbose mode**: Press Ctrl+O to see Claude's reasoning
```

---

## Pillar 2: Session & Context Management - CONFIRMED

### Session Commands Reference

| Command | Purpose |
|---------|---------|
| `/rename <name>` | Name session for later retrieval |
| `claude --continue` / `-c` | Continue most recent conversation |
| `claude --resume` / `-r` | Interactive session picker |
| `/resume` | Switch conversations inside Claude |
| `/context` | View current token usage |
| `/clear` | Full context reset |
| `/compact [instructions]` | Summarize & continue with custom focus |
| `/cost` | Display session usage statistics |

### Checkpoint System

| Action | Effect |
|--------|--------|
| `/rewind` | Open checkpoint interface |
| `Esc` twice | Quick access to checkpoints |
| Code-only restore | Revert files, keep conversation |
| Conversation-only | Reset context, keep code |
| Full restore | Revert both |

**Critical Limitation**: Bash commands (`rm`, `mv`, `cp`) are NOT tracked

### Context Limits - MYTH DEBUNKED

- **"60% rule"**: NOT officially documented
- **Auto-compact trigger**: ~95% capacity (not 60%)
- **Recommended manual compact**: ~70% capacity
- **Token savings**: 50-70% reduction with good CLAUDE.md + strategic `/clear`

### Document & Clear Pattern
```markdown
## Document & Clear Pattern
For complex multi-phase tasks:
1. Have Claude dump plan/progress to a `.md` file
2. Run `/clear` to reset context
3. Start new session: "Read plan.md and continue from step 3"
```

---

## Pillar 3: MCP Server Integration - CONFIRMED

### Configuration Format

**Location hierarchy:**
1. `.mcp.json` (project root) - shared via git
2. `~/.config/claude/mcp.json` - local machine
3. `~/.claude/mcp.json` - global user

**Schema:**
```json
{
  "mcpServers": {
    "server-name": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-name"],
      "env": {
        "API_KEY": "value"
      }
    }
  }
}
```

### High-Value MCP Servers

| Server | Purpose | Package |
|--------|---------|---------|
| GitHub | PR/issue management | `@modelcontextprotocol/server-github` |
| Playwright | Browser automation (replaces Puppeteer) | `@anthropic-ai/playwright-mcp` |
| PostgreSQL | Database queries | `@modelcontextprotocol/server-postgres` |
| Filesystem | Enhanced file ops | `@modelcontextprotocol/server-filesystem` |
| Fetch | Web content retrieval | `@modelcontextprotocol/server-fetch` |
| Memory | Persistent knowledge graph | `@modelcontextprotocol/server-memory` |

### CLI Commands
```bash
claude mcp add <name> -- <command>     # Add server
claude mcp list                         # List configured servers
claude mcp remove <name>                # Remove server
/mcp                                    # Verify connections (inside Claude)
--mcp-debug                            # Debug mode
```

### Context Impact
- Each MCP server adds tool definitions (~50-1000 tokens each)
- Disable unused servers: `/mcp` → disable
- Use project-scoped `.mcp.json` to load only needed servers

---

## Pillar 4: Skills Architecture - CONFIRMED

### Directory Structure
```
.claude/skills/<skill-name>/
├── SKILL.md           # Required: frontmatter + instructions
├── resources/         # Optional: docs loaded on-demand
├── scripts/           # Optional: executable helpers
└── templates/         # Optional: structured prompts
```

### SKILL.md Format
```markdown
---
name: skill-name
description: "Use when the user asks to [specific trigger]. This skill [what it does]."
allowed-tools: Read,Write,Bash,Grep
version: 1.0.0
model: claude-opus-4-5-20251101
---

# Skill Name

## Purpose
One-sentence description.

## When to Use
- User asks for [X]
- Task involves [Y]

## Steps
1. First step
2. Second step

## Output Format
Specify expected output structure.
```

### Auto-Discovery Rules
- Claude reads only `name` and `description` at startup (~100 tokens)
- Full SKILL.md loads only when triggered (~5k max)
- **Critical**: All trigger logic must be in `description` field
- Use "Use when..." format for reliable triggering
- Third-person singular voice

### Decision Matrix

| Feature | Always Loaded | Separate Context | Manual Trigger |
|---------|---------------|------------------|----------------|
| CLAUDE.md | Yes | No | No |
| Slash Commands | No | No | Yes (`/`) |
| Subagents | No | Yes | Delegated |
| Skills | No (metadata only) | No | Auto |

### Activation Rates (Research)
- Simple instructions: ~20% activation
- Forced eval hook: **84% activation** (best)
- LLM eval hook: 80% activation

---

## Pillar 5: Modular CLAUDE.md & Rules - CONFIRMED

### `.claude/rules/` Directory (Official v2.0.64+)

**Structure:**
```
.claude/
├── CLAUDE.md              # Lean overview (<60 lines)
└── rules/
    ├── code-style.md      # Universal code standards
    ├── testing.md         # Test requirements
    ├── security.md        # Security guidelines
    └── frontend/
        └── react.md       # Path-scoped React rules
```

### Path-Scoped Rules
```yaml
---
paths:
  - src/api/**/*.ts
  - src/services/**/*.ts
---

# API Development Rules
- All endpoints must validate input with Zod
- Use OpenAPI decorators
```

Rules only activate when working on matching files.

### `@path/to/import` Syntax
```markdown
# In CLAUDE.md
See @README for project overview
For API patterns, see @docs/api-design.md
@~/.claude/code-standards.md
```

- Inline expansion on load
- Recursive support (max 5 hops)
- Code block exclusion (backtick-wrapped paths not imported)

### Size Guidelines
- **Instruction limit**: ~150-200 instructions reliably followed
- **Claude Code overhead**: ~50 built-in instructions
- **CLAUDE.md budget**: 50-100 instructions remaining
- **Recommended size**: <60 lines in root CLAUDE.md
- **Rule files**: <500 lines each

### Lean CLAUDE.md Template
```markdown
# Project: [Name]

## What
- **Stack**: [Tech stack]
- **Structure**: [Directory layout]

## How
1. **Testing**: `npm test`
2. **Commits**: Conventional Commits
3. **Workflow**: Feature branch → PR → CI

## Rules
- @.claude/rules/code-style.md
- @.claude/rules/testing.md
```

---

## Pillar 6: Visual & Headless Workflows - CONFIRMED

### Visual Iteration Workflow

**Image Input Methods:**
1. **File path**: `"Analyze /path/to/mockup.png"`
2. **Ctrl+V**: Paste from clipboard
3. **Drag-drop**: May need Shift key

**Puppeteer → Playwright Migration:**
- Official `@modelcontextprotocol/server-puppeteer` is **deprecated**
- Use **Playwright MCP** for browser automation

**Visual Loop:**
1. Provide design mock
2. Request implementation
3. Screenshot result (Playwright MCP)
4. Compare & iterate
5. "Make it aesthetically pleasing" trigger

### Headless Mode

**Basic Usage:**
```bash
claude -p "Your task" --output-format json
cat file.txt | claude -p "Analyze this"
```

**Key Flags:**
| Flag | Purpose |
|------|---------|
| `-p` / `--print` | Enable headless mode |
| `--output-format` | text/json/stream-json |
| `--allowedTools` | Pre-approve tools |
| `--append-system-prompt` | Add instructions |
| `--dangerously-skip-permissions` | YOLO mode (use in containers) |
| `--max-turns` | Limit Claude invocations |

### CI/CD Patterns

**Fanning Out:**
```bash
for file in $(find . -name "*.ts"); do
  claude -p "Migrate $file" --allowedTools "Edit" --output-format json
done
```

**Pipelining:**
```bash
cat error.log | claude -p "Analyze errors" | jq '.issues[]'
```

### GitHub Actions
```yaml
- uses: anthropics/claude-code-action@v1
  with:
    anthropic_api_key: ${{ secrets.ANTHROPIC_API_KEY }}
    prompt: "Review this PR"
    claude_args: "--max-turns 5"
```

### Safe YOLO Mode
```bash
docker run --network none \
  -e ANTHROPIC_API_KEY="..." \
  image claude -p "task" \
  --dangerously-skip-permissions \
  --allowedTools "Edit,Bash(git:*)"
```

---

## Pillar 7: Multi-Claude Orchestration - CONFIRMED

### Git Worktrees Setup
```bash
# Create worktrees
git worktree add .worktrees/feature-a -b feature-a main
git worktree add .worktrees/feature-b -b feature-b main

# Run parallel sessions
cd .worktrees/feature-a && claude &
cd .worktrees/feature-b && claude &

# Cleanup
git worktree remove .worktrees/feature-a
```

### Orchestration Patterns

**Pattern 1: Writer + Reviewer**
```
.worktrees/
├── feature/    (Writer agent)
└── review/     (Reviewer agent - fresh context)
```

**Pattern 2: Specialized Teams**
```
Agent 1-2: Components (parallel)
Agent 3-4: Tests (after components)
Agent 5: Documentation
Agent 6: Benchmarks
```

**Pattern 3: Parallel Subagents**
```
"Explore using 4 tasks in parallel:
- Agent 1: src/components/
- Agent 2: src/services/
- Agent 3: tests/
- Agent 4: docs/"
```

### Parallelism Limits
- **Default cap**: 10 concurrent tasks
- **Context**: Each subagent gets isolated 200k window
- **Overflow**: Additional tasks queue automatically

### Conflict Mitigation
1. **Directory partitioning**: Each agent owns specific directories
2. **Communication ledger**: Shared `PROGRESS.md` file
3. **Commit messages**: Agents read each other's commits
4. **Sequential dependencies**: Tests wait for components

### Decision Guide

| Use Parallel When | Use Sequential When |
|-------------------|---------------------|
| Independent tasks | Later depends on earlier |
| Low conflict risk | High conflict risk |
| >3 hours saved | <1 hour saved |
| Fresh context helps | Context continuity matters |

---

## Implementation Priority

### Phase 1: Quick Wins (1-2 hours)
1. Add thinking triggers section to CLAUDE.md
2. Add session management section to CLAUDE.md
3. Create `.claude/rules/` directory structure

### Phase 2: Core Features (2-4 hours)
4. Create `.mcp.json.template` with common servers
5. Create 3 initial skills (tdd, code-review, security-review)
6. Refactor CLAUDE.md to <60 lines with @imports

### Phase 3: Advanced (4-6 hours)
7. Add headless mode documentation
8. Document multi-agent orchestration patterns
9. Create visual workflow command

### Verification Checklist
- [ ] Thinking triggers documented with examples
- [ ] Session commands reference complete
- [ ] `.claude/rules/` directory exists with path-scoped rules
- [ ] `.mcp.json.template` created
- [ ] At least 3 skills in `.claude/skills/`
- [ ] CLAUDE.md refactored to <60 lines
- [ ] Headless mode patterns documented
- [ ] Multi-agent patterns documented

---

## Sources Summary

Research compiled from:
- [Anthropic: Claude Code Best Practices](https://www.anthropic.com/engineering/claude-code-best-practices)
- [Claude Code Official Documentation](https://code.claude.com/docs/)
- [alexop.dev: Claude Code Customization Guide](https://alexop.dev/posts/claude-code-customization-guide-claudemd-skills-subagents/)
- [Model Context Protocol Official](https://modelcontextprotocol.io/)
- [PulseMCP Server Registry](https://www.pulsemcp.com/servers) (7,520+ servers)
- [GitHub: anthropics/claude-code-action](https://github.com/anthropics/claude-code-action)
- [Various community guides and GitHub issues]

---

*Synthesis completed: 2026-01-04*
*Status: Ready for Implementation*
