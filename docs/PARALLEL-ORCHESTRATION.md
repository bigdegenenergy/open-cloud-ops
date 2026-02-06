# Parallel Agent Orchestration Guide

The "Boris Setup" is predicated on overcoming the inherent constraints of LLMs: latency and context seriality. By running five or more concurrent Claude Code sessions, the developer shifts from a synchronous "write-compile-debug" loop to an asynchronous "dispatch-review-merge" loop.

## The Paradigm Shift

**From Copilot to Orchestrator:**

| Old Model (Copilot) | New Model (Orchestrator) |
|---------------------|-------------------------|
| AI assists single developer | Developer manages AI team |
| Linear execution | Parallel execution |
| Wait for completion | Dispatch and review |
| Human writes code | Human reviews code |
| Cognitive bottleneck | Cognitive freedom |

## The Multi-Worktree Architecture

A single git checkout is insufficient for parallel agents. Multiple agents attempting to modify files, switch branches, or run build processes in the same directory will lead to:
- Race conditions
- Git lock contention
- LSP confusion

### Solution: Git Worktrees

Unlike cloning a repository multiple times (which duplicates the object database), `git worktree` allows multiple working directories to share a single `.git` history.

```bash
# Create the project swarm structure
mkdir ~/project-swarm
cd ~/project-swarm

# Clone the main repository
git clone git@github.com:org/project.git main
cd main

# Create isolated worktrees for each agent
git worktree add ../agent-1 -b feature/agent-1
git worktree add ../agent-2 -b feature/agent-2
git worktree add ../agent-3 -b feature/agent-3
git worktree add ../agent-4 -b feature/agent-4
git worktree add ../agent-5 -b feature/agent-5
```

### Resulting Directory Structure

```
~/project-swarm/
├── main/          # Orchestrator workspace (Tab 1)
├── agent-1/       # Feature A (Tab 2)
├── agent-2/       # Feature B (Tab 3)
├── agent-3/       # Testing (Tab 4)
├── agent-4/       # Infrastructure (Tab 5)
└── agent-5/       # Documentation (Tab 6)
```

Each worktree has:
- Its own working directory
- Its own `node_modules` / dependencies
- Its own build artifacts
- Its own `.env` files if needed

But they all share:
- The same `.git` object database
- The same remote refs
- The same history

## Terminal Management Strategy

Use a terminal multiplexer (tmux, iTerm2, Windows Terminal) with numbered tabs:

| Tab | Worktree | Role | Purpose |
|-----|----------|------|---------|
| 1 | main/ | Orchestrator | Planning, reviewing, merging |
| 2 | agent-1/ | Implementer | Feature A development |
| 3 | agent-2/ | Implementer | Feature B development |
| 4 | agent-3/ | QA Engineer | Testing and verification |
| 5 | agent-4/ | DevOps | Infrastructure and CI/CD |

## The Workflow in Action

### 1. Initialization (Tab 1 - Orchestrator)

```bash
# In the main worktree
claude
> /plan "Implement OAuth2 authentication"
```

Agent generates PLAN.md. Human reviews and approves.

### 2. Dispatch (Tabs 2-5)

Switch to each agent tab and dispatch tasks:

**Tab 2 (Feature A):**
```bash
cd ~/project-swarm/agent-1
claude
> "Implement OAuth2 user model changes (Phase 1 of PLAN.md)"
```

**Tab 3 (Feature B):**
```bash
cd ~/project-swarm/agent-2
claude
> "Update frontend login component with OAuth buttons"
```

**Tab 4 (QA):**
```bash
cd ~/project-swarm/agent-3
claude
> /test-driven "Write e2e tests for OAuth login flow"
```

### 3. Asynchronous Work

The human is now free. All 4 agents work in parallel on isolated worktrees.

Use notification hooks (Stop hook) to know when agents finish:
```bash
# Agent finished → Desktop notification appears
# Human switches to that tab to review
```

### 4. Review and Merge Cycle

When Agent 3 finishes:
1. Switch to Tab 3
2. Run `/review` to get critical feedback
3. Iterate if needed
4. Commit changes in the worktree

As Orchestrator (Tab 1):
1. Merge branches from agent worktrees
2. Resolve any architectural conflicts
3. Push to main

## Configuration Hierarchy

Claude Code uses a hierarchical configuration system:

### 1. Global Settings (`~/.claude/settings.json`)
Personal preferences that apply to ALL projects:
- Notification sounds
- Personal slash commands
- Default model preferences

### 2. Project Settings (`.claude/settings.json`)
Shared team configuration (committed to git):
- Allowed commands
- Hook configurations
- MCP servers
- Team slash commands

### 3. Local Settings (`.claude/settings.local.json`)
Machine-specific overrides (gitignored):
- Absolute paths
- Local environment overrides
- Debug configurations

## Best Practices for Parallel Work

### 1. Clear Task Boundaries
Each agent should work on isolated files/features to minimize merge conflicts.

### 2. Frequent Fetching
When Agent 1 fetches, all agents have access to those objects. Keep synced.

### 3. Human as Merge Point
Never have agents merge each other's work. The human resolves conflicts.

### 4. Use Notifications
Configure Stop hooks to alert when agents complete. Don't stare at terminals.

### 5. Role-Based Worktrees
Consider naming worktrees by role (frontend/, backend/, infra/) if agents are permanently assigned.

## tmux Session Setup Script

```bash
#!/bin/bash
# setup-swarm.sh - Create tmux session for 5-agent workflow

SESSION="swarm"

tmux new-session -d -s $SESSION -n "main" -c ~/project-swarm/main
tmux new-window -t $SESSION -n "agent-1" -c ~/project-swarm/agent-1
tmux new-window -t $SESSION -n "agent-2" -c ~/project-swarm/agent-2
tmux new-window -t $SESSION -n "agent-3" -c ~/project-swarm/agent-3
tmux new-window -t $SESSION -n "agent-4" -c ~/project-swarm/agent-4

tmux attach -t $SESSION
```

## The Force Multiplier

This architecture effectively multiplies the output of a single senior engineer by:
1. Decoupling thought from execution
2. Enabling true parallel work
3. Providing isolated environments
4. Automating the review cycle

The human moves from individual contributor to Engineering Manager, orchestrating a team of capable synthetic developers.
