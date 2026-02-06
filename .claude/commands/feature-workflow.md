---
description: Full-stack feature development workflow. Orchestrates multiple agents for complete feature implementation.
---

# Full-Stack Feature Development Workflow

You are orchestrating a complete feature development workflow that coordinates multiple specialized agents.

## Workflow Phases

This workflow follows a structured approach to feature development:

### Phase 1: Planning
First, understand and plan the feature:
1. Analyze requirements with the user
2. Identify affected components (frontend, backend, database)
3. Design the implementation approach
4. Consider edge cases and error handling

### Phase 2: Backend Development
Invoke the appropriate agents for backend work:
- Use `@backend-architect` for API design decisions
- Use `@database-architect` for schema design
- Use `@python-pro` or `@typescript-pro` for implementation
- Use `@security-auditor` for security review

### Phase 3: Frontend Development (if needed)
For frontend components:
- Use `@frontend-specialist` for UI implementation
- Ensure accessibility compliance
- Implement proper error states

### Phase 4: Testing
Comprehensive test coverage:
- Use `@test-automator` to create test suites
- Unit tests for business logic
- Integration tests for APIs
- E2E tests for critical paths

### Phase 5: Review & Polish
Final quality checks:
- Use `@code-reviewer` for code review
- Use `@code-simplifier` to clean up
- Ensure documentation is updated

### Phase 6: Deployment
Prepare for deployment:
- Use `@infrastructure-engineer` for deployment config
- Use `@verify-app` for final verification

## Current Task

**Git Status:** !`git status -sb`

**Recent Changes:** !`git log --oneline -5`

## Instructions

1. Start by gathering requirements from the user
2. Create a plan before implementing
3. Use the appropriate agents for each phase
4. Run tests after each significant change
5. Get user approval before proceeding to next phase

## Example Agent Invocations

```
# For architecture decisions
Invoke @backend-architect to design the API structure

# For database changes
Invoke @database-architect to design the schema

# For implementation
Invoke @python-pro to implement the backend service

# For testing
Invoke @test-automator to create comprehensive tests

# For review
Invoke @code-reviewer to review the changes
```

**Important**: Keep the user informed of progress and get approval at key decision points.
