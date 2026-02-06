# Gap Fill Implementation Plan

> **Objective:** Enhance the bigdegenenergy/ai-dev-toolkit repository to match and exceed industry best practices as documented in Joe Njenga's "17 Best Claude Code Workflows" article.

---

## Executive Summary

This plan addresses **5 Major Gaps**, **5 Medium Gaps**, and **4 Minor Gaps** identified in the gap analysis. The implementation is organized into **7 Pillars**, each representing a cohesive area of functionality.

**Timeline:** Implementation-ready within this session
**Approach:** Research-first, then implement with verification

---

## Pillar 1: Extended Thinking & Cognitive Modes

### Gap Severity: **MAJOR**

### Current State

- No documentation of thinking triggers
- No mention of thinking budget levels
- Verbose mode (`Ctrl+O`) undocumented

### Target State

Document the full thinking spectrum with use-case guidance.

> **Note:** These triggers are **native to Claude Code CLI** (preprocessed before API calls). No implementation code is needed—only documentation.

| Trigger                       | Budget         | Use Case                                         |
| ----------------------------- | -------------- | ------------------------------------------------ |
| `think`                       | ~4,000 tokens  | Refactoring, simple fixes, error handling        |
| `think hard`                  | ~10,000 tokens | Caching strategy, migration planning, API design |
| `think harder` / `ultrathink` | ~32,000 tokens | Major architecture, comprehensive security audit |

**Important:** These triggers are case-insensitive and work anywhere in the prompt. They do NOT work in claude.ai web interface or direct API calls—only Claude Code CLI.

### Implementation Tasks

1. **Update CLAUDE.md** (documentation only—triggers are built-in)
   - Add "Thinking Triggers" section
   - Document when to use each level
   - Add anti-pattern: "Don't use ultrathink for everything"

2. **Create `.claude/rules/thinking.md`**
   - Detailed guidance on cognitive modes
   - Examples of appropriate prompts for each level
   - Cost/benefit analysis

3. **Update `/plan` command**
   - Suggest appropriate thinking level based on task complexity
   - Add "think hard" trigger for architecture planning

### Files to Create/Modify

- `CLAUDE.md` (add section)
- `.claude/rules/thinking.md` (new)
- `.claude/commands/plan.md` (update)

---

## Pillar 2: Session & Context Management

### Gap Severity: **MAJOR**

### Current State

- No session management documentation
- No context monitoring guidance
- No checkpoint/rewind documentation
- "Document & Clear" pattern unknown

### Target State

Full session lifecycle management with context hygiene practices.

### Key Concepts to Document

#### Session Commands

| Command                    | Purpose                                          |
| -------------------------- | ------------------------------------------------ |
| `/rename <name>`           | Name current session for later retrieval         |
| `claude --continue` / `-c` | Continue most recent conversation                |
| `claude --resume` / `-r`   | Interactive session picker                       |
| `/resume`                  | Switch to different conversation (inside Claude) |
| `/context`                 | Check current token usage                        |
| `/clear`                   | Clear context for fresh start                    |

#### Checkpoint System

| Action                    | What It Does                     |
| ------------------------- | -------------------------------- |
| `/rewind`                 | Access checkpoint system         |
| `Esc` twice               | Quick access to checkpoints      |
| Code-only restore         | Revert files, keep conversation  |
| Conversation-only restore | Reset context, keep code changes |
| Full restore              | Revert both to prior point       |

#### The "Document & Clear" Pattern

For complex multi-phase tasks:

1. Have Claude dump current plan/progress to a `.md` file
2. Run `/clear` to reset context
3. Start new session: "Read `plan.md` and continue from step 3"

#### Context Hygiene Rules

> **Note:** The "60% rule" is a myth. Auto-compact triggers at ~95% capacity.

- Run `/compact` manually at **~70% capacity** (before auto-compact kicks in)
- Auto-compact triggers at **~95%** (not 60%)—by then you've lost control over what gets summarized
- Run `/context` periodically during long sessions to monitor usage
- Clear context between workflow phases with `/clear`
- Disable unused MCP servers before compaction: `/mcp`

### Implementation Tasks

1. **Update CLAUDE.md**
   - Add "Session Management" section
   - Add "Context Hygiene" section
   - Document the ~70% manual compact recommendation (NOT the debunked 60% rule)

2. **Create `.claude/rules/context.md`**
   - Detailed context management strategies
   - Warning signs of context bloat
   - Recovery procedures

3. **Create `.claude/commands/checkpoint.md`** (optional)
   - Quick reference for checkpoint operations

4. **Update Standard Workflow**
   - Add `/clear` between phases
   - Add context check reminders

### Files to Create/Modify

- `CLAUDE.md` (add sections)
- `.claude/rules/context.md` (new)
- `.claude/commands/checkpoint.md` (optional, new)

---

## Pillar 3: MCP Server Integration

### Gap Severity: **MAJOR**

### Current State

- No MCP configuration
- No `.mcp.json` file
- No documentation on MCP servers

### Target State

Template MCP configuration with documentation for common integrations.

### High-Value MCP Integrations

| Server         | Use Case                             | Package                                   |
| -------------- | ------------------------------------ | ----------------------------------------- |
| **GitHub**     | Issue/PR management, code review     | `@modelcontextprotocol/server-github`     |
| **Playwright** | Visual testing, screenshot workflows | `@anthropic-ai/playwright-mcp`            |
| **Filesystem** | Enhanced file operations             | `@modelcontextprotocol/server-filesystem` |
| **PostgreSQL** | Database queries and exploration     | `@modelcontextprotocol/server-postgres`   |
| **Sentry**     | Error analysis and debugging         | `@modelcontextprotocol/server-sentry`     |
| **Figma**      | Design-to-code workflows             | Community server                          |
| **Slack**      | Team notifications                   | `@modelcontextprotocol/server-slack`      |

> **Note:** The official `@modelcontextprotocol/server-puppeteer` is **deprecated**. Use Playwright MCP instead for browser automation.

### Implementation Tasks

1. **Create `.mcp.json.template`**
   - Template with common MCP servers
   - Comments explaining each integration
   - Instructions for customization

2. **Update CLAUDE.md**
   - Add "MCP Servers" section
   - Document setup process
   - Link to template

3. **Create `docs/SETUP-MCP.md`**
   - Detailed MCP setup guide
   - Server-specific configuration
   - Troubleshooting tips

4. **Add MCP management to hooks**
   - Consider SessionStart hook for MCP status
   - Document `/mcp` command usage

### Files to Create/Modify

- `.mcp.json.template` (new)
- `CLAUDE.md` (add section)
- `docs/SETUP-MCP.md` (new)

---

## Pillar 4: Skills Architecture

### Gap Severity: **MAJOR**

### Current State

- No `.claude/skills/` directory
- No SKILL.md files
- Skills feature completely unused

### Target State

Implement Skills for auto-discovered, context-aware expertise.

### Understanding Skills vs Commands vs Agents

| Feature            | Trigger         | Context  | Best For             |
| ------------------ | --------------- | -------- | -------------------- |
| **CLAUDE.md**      | Always loaded   | Main     | Project conventions  |
| **Slash Commands** | Manual `/`      | Main     | Explicit workflows   |
| **Subagents**      | Delegated       | Separate | Research-heavy tasks |
| **Skills**         | Auto-discovered | Main     | Domain expertise     |

### Skills to Implement

1. **`tdd/SKILL.md`** - Test-driven development patterns
2. **`security-review/SKILL.md`** - Security audit checklists
3. **`pr-review/SKILL.md`** - PR review standards
4. **`refactoring/SKILL.md`** - Safe refactoring patterns
5. **`api-design/SKILL.md`** - REST/GraphQL best practices
6. **`debugging/SKILL.md`** - Systematic debugging approach

### Skill Structure

```
.claude/skills/<skill-name>/
├── SKILL.md           # Main definition (required)
├── PATTERNS.md        # Common patterns (optional)
├── CHECKLIST.md       # Quality checklist (optional)
└── scripts/           # Helper scripts (optional)
    └── validate.sh
```

### SKILL.md Format

```markdown
---
name: skill-name
description: When to auto-trigger this skill
---

# Skill Title

## Critical Steps

1. First step
2. Second step

## Patterns

- Pattern A
- Pattern B

## Anti-Patterns

- Don't do X
- Avoid Y
```

### Implementation Tasks

1. **Create `.claude/skills/` directory**

2. **Implement Priority Skills**
   - `tdd/SKILL.md`
   - `security-review/SKILL.md`
   - `pr-review/SKILL.md`

3. **Update CLAUDE.md**
   - Add "Skills" section
   - Explain auto-discovery mechanism
   - Document when skills are triggered

4. **Create `docs/SKILLS.md`**
   - How to create custom skills
   - Skill vs Command decision guide
   - Examples and templates

### Files to Create/Modify

- `.claude/skills/tdd/SKILL.md` (new)
- `.claude/skills/security-review/SKILL.md` (new)
- `.claude/skills/pr-review/SKILL.md` (new)
- `CLAUDE.md` (add section)
- `docs/SKILLS.md` (new)

---

## Pillar 5: Modular CLAUDE.md & Rules

### Gap Severity: **MEDIUM**

### Current State

- CLAUDE.md is 300+ lines (exceeds 50-line guideline for lean projects)
- No `.claude/rules/` directory
- `@path/to/import` syntax not used

### Target State

Modular, maintainable configuration with lean CLAUDE.md.

### The Problem with Monolithic CLAUDE.md

- Frontier LLMs follow ~150-200 instructions consistently
- Claude Code's system prompt already contains ~50 instructions
- As instruction count increases, ALL instructions are followed less consistently

### Modular Structure

```
.claude/
├── rules/
│   ├── code-style.md      # Formatting, naming, linting
│   ├── testing.md         # Test patterns, coverage
│   ├── security.md        # Security requirements
│   ├── git.md             # Commit format, branching
│   ├── typescript.md      # TS-specific rules
│   └── documentation.md   # Doc standards
```

### `@path/to/import` Syntax

Instead of embedding content:

```markdown
## Database Patterns

For Dexie.js usage, see @docs/database-patterns.md
```

Claude loads the file only when needed, preserving context.

### Implementation Tasks

1. **Create `.claude/rules/` directory**
   - Extract sections from CLAUDE.md into modular files
   - Keep each file focused (<30 lines ideal)

2. **Refactor CLAUDE.md**
   - Reduce to <50 core lines
   - Use `@path/to/file` references
   - Keep only essential, always-needed info

3. **Document the pattern**
   - Add explanation in CLAUDE.md
   - Create template for new rules

### Files to Create/Modify

- `.claude/rules/code-style.md` (new)
- `.claude/rules/testing.md` (new)
- `.claude/rules/security.md` (new)
- `.claude/rules/git.md` (new)
- `CLAUDE.md` (refactor)

---

## Pillar 6: Visual & Headless Workflows

### Gap Severity: **MEDIUM**

### Current State

- Visual iteration workflow undocumented
- Headless mode (`-p` flag) undocumented
- CI/CD patterns implicit in GitHub Actions but not explained

### Target State

Document both visual (design-to-code) and headless (automation) workflows.

### Visual Iteration Workflow

1. Provide design mock (paste, drag-drop, or file path)
2. Ask Claude to implement
3. Use **Playwright MCP** to screenshot result (Puppeteer MCP is deprecated)
4. Compare and iterate
5. Tip: "Make it aesthetically pleasing"

### Headless Mode Patterns

#### Basic Usage

```bash
# Simple one-shot
claude -p "Update copyright headers to 2025" --json

# With stdin
cat src/utils.ts | claude -p "Find bugs"

# With specific permissions
claude -p "Fix failing test" \
  --allow-tools Edit,View,Bash \
  --output-format json
```

#### Two Primary Patterns

1. **Fanning Out**: Generate task list, loop calling Claude for each
2. **Pipelining**: `cat error.txt | claude -p 'explain' > output.txt`

### Implementation Tasks

1. **Create `.claude/commands/visual-iterate.md`**
   - Document screenshot workflow
   - Require Playwright MCP (NOT Puppeteer—deprecated)

2. **Update CLAUDE.md**
   - Add "Headless Mode" section
   - Document `-p` flag and options
   - Add "Visual Workflow" section

3. **Create `docs/HEADLESS-MODE.md`**
   - Detailed headless mode guide
   - CI/CD integration patterns
   - GitHub Actions examples

4. **Update existing GitHub Actions**
   - Add comments explaining headless patterns
   - Document `--allow-tools` usage

### Files to Create/Modify

- `.claude/commands/visual-iterate.md` (new, optional)
- `CLAUDE.md` (add sections)
- `docs/HEADLESS-MODE.md` (new)

---

## Pillar 7: Multi-Claude Orchestration

### Gap Severity: **MEDIUM**

### Current State

- Git worktrees mentioned but not detailed
- No Writer + Reviewer pattern
- No Specialized Teams pattern
- Parallel subagent invocation undocumented

### Target State

Document advanced multi-agent orchestration patterns.

### Pattern 1: Writer + Reviewer

Simple but effective:

- **Claude A**: Writes code in worktree-1
- **Claude B**: Reviews code in worktree-2 (read-only)

### Pattern 2: Specialized Teams

For large refactors:
| Agent | Role |
|-------|------|
| Agent 1-2 | Work on different component folders |
| Agent 3-4 | Update tests as components complete |
| Agent 5 | Regenerate documentation |
| Agent 6 | Run performance benchmarks |

### Pattern 3: Parallel Subagent Research

```
"Explore the codebase using 4 tasks in parallel.
Each agent should explore different directories:
- Agent 1: src/components/
- Agent 2: src/services/
- Agent 3: src/utils/
- Agent 4: tests/"
```

### Git Worktree Commands

```bash
# Create isolated worktrees
git worktree add ../project-feature-a -b feature-a main
git worktree add ../project-bugfix -b hotfix main

# Run separate sessions
cd ../project-feature-a && claude
cd ../project-bugfix && claude

# List worktrees
git worktree list

# Remove when done
git worktree remove ../project-feature-a
```

### Implementation Tasks

1. **Update CLAUDE.md**
   - Add "Multi-Claude Orchestration" section
   - Document worktree setup
   - Reference patterns

2. **Create `docs/MULTI-AGENT.md`**
   - Detailed orchestration guide
   - Pattern descriptions with examples
   - Conflict resolution strategies

3. **Create `.claude/commands/parallel-research.md`**
   - Command for parallel subagent exploration

4. **Update subagent documentation**
   - Document parallel invocation syntax
   - Context efficiency explanation

### Files to Create/Modify

- `CLAUDE.md` (add section)
- `docs/MULTI-AGENT.md` (new)
- `.claude/commands/parallel-research.md` (new)

---

## Implementation Order

### Phase 1: Foundation (Highest Impact)

1. **Pillar 1**: Extended Thinking (quick win, high value)
2. **Pillar 2**: Session & Context Management (critical for efficiency)

### Phase 2: Advanced Features

3. **Pillar 4**: Skills Architecture (new capability)
4. **Pillar 3**: MCP Server Integration (extensibility)

### Phase 3: Optimization

5. **Pillar 5**: Modular CLAUDE.md (maintainability)
6. **Pillar 6**: Visual & Headless Workflows (completeness)
7. **Pillar 7**: Multi-Claude Orchestration (power users)

---

## Success Criteria

Each pillar is complete when:

- [ ] All new files created
- [ ] CLAUDE.md updated with relevant sections
- [ ] Documentation is actionable (not just descriptive)
- [ ] Examples are provided where applicable
- [ ] Patterns are explained with use cases

---

## Research Required

Before implementation, research agents should investigate:

1. **Pillar 1**: Exact thinking budget values, Ctrl+O behavior, API cost implications
2. **Pillar 2**: Full session command list, checkpoint limitations, context compaction behavior
3. **Pillar 3**: Available MCP servers, configuration format, setup requirements
4. **Pillar 4**: SKILL.md format specification, auto-discovery rules, skill vs command behavior
5. **Pillar 5**: @import syntax specifics, rules/ directory behavior, instruction limits
6. **Pillar 6**: Playwright MCP setup (Puppeteer deprecated), headless mode output formats, CI/CD best practices
7. **Pillar 7**: Worktree limitations, subagent parallelism limits, conflict patterns

---

_Plan created: 2026-01-04_
_Status: Research Phase_
