---
allowed-tools: Bash(git add:*), Bash(git status:*), Bash(git commit:*), Bash(git push:*), Bash(gh pr create:*)
argument-hint: [commit-message]
description: Commit staged changes, push to the current branch, and create a pull request.
model: haiku
---

## Context for Claude

- **Current git status:** !`git status`
- **Current branch:** !`git branch --show-current`
- **Recent commits:** !`git log --oneline -5`
- **Staged changes:** !`git diff --staged`
- **Unstaged changes:** !`git diff`

## Your Task

1. **Review** the staged changes and the current git status provided in the 'Context' section.
2. **Commit** the staged changes using the provided commit message: "$ARGUMENTS".
   - If no commit message is provided, generate a descriptive one based on the changes.
   - Follow conventional commit format (feat:, fix:, docs:, etc.)
3. **Push** the committed changes to the remote branch.
4. **Create a Pull Request** using the `gh pr create` tool.
   - Use the commit message as the PR title
   - Generate a detailed description based on the changes
   - Include relevant context and testing notes

## Quality Checks

Before completing:

- Verify all tests pass (if applicable)
- Check for any uncommitted changes
- Ensure the PR description is comprehensive

**Note:** If no changes are staged, inform the user and stop. Ensure all steps are executed sequentially and report the outcome of each step.
