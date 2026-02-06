# Bash Aliases for Claude Code Workflow

Add these to your `~/.zshrc` or `~/.bashrc` for faster workflows.

## Core Claude Commands

```bash
# Quick access to Claude Code
alias cc='claude'
alias cci='claude /init'

# Common slash commands
alias cc-plan='claude "/plan"'
alias cc-qa='claude "/qa"'
alias cc-ship='claude "/ship"'
alias cc-test='claude "/test-driven"'
alias cc-review='claude "/review"'
alias cc-metrics='claude "/metrics"'
```

## Git Workflow Shortcuts

```bash
# Git basics
alias gs='git status'
alias gd='git diff'
alias ga='git add'
alias gc='git commit'
alias gp='git push'
alias gl='git log --oneline -20'
alias gb='git branch'
alias gco='git checkout'

# Git with Claude
alias gcp='git add -A && claude "/ship"'  # Stage all and ship
alias gcr='claude "/review"'               # Review before commit
```

## Parallel Development

```bash
# Create worktree for parallel development
worktree-add() {
    local name=$1
    local branch=${2:-$1}
    git worktree add "../$name" -b "$branch" 2>/dev/null || \
    git worktree add "../$name" "$branch"
    echo "Worktree created at ../$name"
    echo "Run: cd ../$name && claude"
}

# Remove worktree
worktree-rm() {
    local name=$1
    git worktree remove "../$name" --force
    echo "Worktree removed: $name"
}

# List all worktrees
alias worktrees='git worktree list'
```

## Multi-Agent Workflow

```bash
# Launch multiple Claude sessions (macOS with iTerm2)
claude-team() {
    local count=${1:-5}
    local repo=$(pwd)

    for i in $(seq 1 $count); do
        osascript <<EOF
tell application "iTerm"
    create window with default profile
    tell current session of current window
        write text "cd $repo && claude"
    end tell
end tell
EOF
        sleep 0.5
    done
    echo "Launched $count Claude sessions"
}

# Launch Claude in tmux panes
claude-tmux() {
    local count=${1:-4}
    tmux new-session -d -s claude

    for i in $(seq 2 $count); do
        tmux split-window -h
        tmux select-layout tiled
    done

    for i in $(seq 0 $((count-1))); do
        tmux send-keys -t claude:0.$i "claude" Enter
    done

    tmux attach -t claude
}
```

## Quality Checks

```bash
# Run all quality checks
alias check-all='npm run lint && npm run type-check && npm test'

# Quick test
alias t='npm test'
alias tw='npm run test:watch'
alias tc='npm run test:coverage'

# Lint and format
alias lint='npm run lint'
alias fix='npm run lint:fix && npm run format'
```

## Project Navigation

```bash
# Quick navigation (customize paths)
alias proj='cd ~/projects'
alias work='cd ~/work'

# Jump to common directories
alias src='cd ./src'
alias tests='cd ./tests'
alias docs='cd ./docs'
```

## Claude Session Management

```bash
# Clear Claude context
alias cc-clear='rm -rf .claude/.context 2>/dev/null; echo "Context cleared"'

# Reset Claude state
alias cc-reset='rm -rf .claude/.state 2>/dev/null; echo "State reset"'

# View Claude logs (if logging enabled)
alias cc-logs='cat .claude/metrics/*.csv 2>/dev/null | tail -20'
```

## Deployment Shortcuts

```bash
# Deploy to staging
alias deploy-staging='claude "/deploy-staging"'

# Quick build
alias build='npm run build'

# Build and deploy
alias ship='npm run build && npm run deploy'
```

## Investigation Helpers

```bash
# Find TODOs in codebase
alias todos='grep -r "TODO\|FIXME\|HACK\|XXX" --include="*.ts" --include="*.tsx" --include="*.js" .'

# Find console.logs
alias find-logs='grep -r "console.log" --include="*.ts" --include="*.tsx" .'

# Count lines of code
alias loc='find . -name "*.ts" -o -name "*.tsx" | xargs wc -l | tail -1'
```

## Installation

1. Copy the aliases you want to your shell config:
   ```bash
   # For zsh
   nano ~/.zshrc

   # For bash
   nano ~/.bashrc
   ```

2. Paste the aliases at the end of the file

3. Reload your shell:
   ```bash
   source ~/.zshrc  # or ~/.bashrc
   ```

4. Test with:
   ```bash
   cc --version
   ```

## Tips

- Customize paths and commands to match your project
- Add project-specific aliases as needed
- Use functions for complex multi-step operations
- Keep aliases short but memorable

---

**See also:**
- [PARALLEL-ORCHESTRATION.md](./PARALLEL-ORCHESTRATION.md) for multi-agent workflows
- Setup script for automatic installation
