# Team Documentation for Claude Code

This file contains team-specific knowledge, patterns, and conventions that Claude should follow when working on this codebase. Update this file weekly as patterns emerge.

## The Virtual Team

### Quick Reference - Commands

| Role | Command | When to Use |
|------|---------|-------------|
| Architect | `/plan` | Before complex features |
| QA Engineer | `/qa` | When tests need fixing |
| TDD | `/test-driven` | Red-green-refactor loop |
| Gatekeeper | `/test-and-commit` | Run tests, only commit if pass |
| Reviewer | `/review` | Code review (read-only) |
| Refactorer | `/simplify` | After implementation |
| DevOps | `/ship` | Ready to commit |
| Deploy | `/deploy-staging` | Build and deploy to staging |

### Quick Reference - Agents

| Role | Agent | Specialty |
|------|-------|-----------|
| Code Reviewer | `@code-reviewer` | Critical code review |
| QA | `@verify-app` | End-to-end testing |
| Security | `@security-auditor` | Vulnerability scanning (read-only) |
| Frontend | `@frontend-specialist` | React, TS, accessibility |
| DevOps | `@infrastructure-engineer` | Docker, K8s, CI/CD |
| Cleanup | `@code-simplifier` | Code hygiene |

### Standard Workflow

```
1. /plan           → Think before coding
2. [implement]     → Write the code
3. /simplify       → Clean up
4. /qa             → Verify tests pass
5. /review         → Self-review
6. /ship           → Commit, push, PR
```

### Parallel Workflow (Advanced)

```
Tab 1: /plan        → Generate PLAN.md
Tab 2: Backend      → Dispatch API tasks
Tab 3: Frontend     → Dispatch UI tasks
Tab 4: QA           → /test-driven
Tab 5: Infra        → CI/CD tasks

[Review outputs as notifications arrive]
Tab 1: Merge all branches
```

## Project Overview

**Description:** Claude Code meta-repository for virtual team configuration

**Architecture:** Configuration-as-code using Claude Code's native features

**Tech Stack:**
- Language: Markdown, Bash, Python (hooks)
- Framework: Claude Code slash commands, hooks, subagents
- Target: Solo developers and small teams

## Code Conventions

### Naming Conventions
- **Files:** Use kebab-case for filenames (e.g., `user-service.ts`)
- **Classes:** Use PascalCase (e.g., `UserService`)
- **Functions:** Use camelCase (e.g., `getUserById`)
- **Constants:** Use UPPER_SNAKE_CASE (e.g., `MAX_RETRY_COUNT`)

### Slash Command Conventions
- Use frontmatter for metadata (description, model, allowed-tools)
- Include pre-computed context with inline bash (`!`command``)
- Assign a clear persona ("You are the **Staff Architect**")
- Be explicit about when to stop and wait for approval

### Hook Conventions
- PostToolUse hooks must fail silently (never block the agent)
- Stop hooks can exit non-zero to alert Claude of issues
- Use Python for complex logic, shell for simple commands
- Log metrics for continuous improvement

## Common Patterns

### Pre-compute Context
Always inject real-time data into slash commands:
```markdown
## Context
- **Git Status:** !`git status -sb`
- **Recent Changes:** !`git diff --stat HEAD~1`
```

### Iterative Loops (QA Pattern)
For testing, use a "keep going until green" pattern:
```markdown
1. Run tests
2. If fail: analyze, fix, goto 1
3. If pass: report and exit
```

### Critical Review Pattern
Use explicit instructions to override agreeable behavior:
```markdown
**Be critical, not agreeable.** Find problems. The team depends on you.
```

## Things Claude Should NOT Do

### Patterns to Avoid
1. **Don't use `any` type in TypeScript** - Always provide specific types
2. **Don't commit commented-out code** - Delete it or use feature flags
3. **Don't hardcode configuration** - Use environment variables
4. **Don't skip error handling** - Every external call needs try-catch
5. **Don't skip the planning phase** - Use `/plan` for complex features

### Common Mistakes
- Skipping tests before committing
- Not running `/simplify` after complex changes
- Force pushing without permission
- Implementing without a plan for complex features

## Things Claude SHOULD Do

### Always Do These
1. **Use `/plan` first** for complex features
2. **Run `/qa` before committing** - Ensure tests pass
3. **Use `/simplify`** after complex changes
4. **Use `@code-reviewer`** for self-review
5. **Write meaningful commit messages** - Follow conventional commits

### Best Practices
- Pre-compute context in slash commands with inline bash
- Include "be critical" in review prompts
- Log important operations and errors
- Update this docs.md when new patterns emerge

## Team-Specific Knowledge

### Known Issues
- PostToolUse hooks require the formatter to be installed (fail silently if not)
- Stop hooks may timeout on large test suites

### Performance Considerations
- Formatters run on every file edit (keep them fast)
- QA loops have a max iteration limit to prevent infinite loops

### Security Notes
- Never commit .env files or secrets
- Always validate inputs at system boundaries
- Use allowed-tools in frontmatter to restrict dangerous operations

## Update Log

Track when this document is updated and why:

- **2025-01-03**: Initial documentation created
- **2025-01-03**: Added virtual team quick reference
- **2025-01-03**: Added slash command and hook conventions

---

**Note:** This is a living document. Update it whenever you discover something Claude should know about this codebase.
