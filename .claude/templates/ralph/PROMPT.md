# Development Instructions

> Project-specific guidance for autonomous development.

## Project Overview

[Describe what this project does and its goals]

## Requirements

[List the key requirements to implement]

1. Requirement 1
2. Requirement 2
3. Requirement 3

## Technical Constraints

- Language/Framework: [e.g., TypeScript, Python, Go]
- Testing Framework: [e.g., Jest, pytest, Go test]
- Build System: [e.g., npm, pip, make]

## Directory Structure

```
project/
├── src/           # Source code
├── tests/         # Test files
├── docs/          # Documentation
└── fix_plan.md   # Task tracking
```

## Development Rules

1. **ONE task per loop** - Focus on completing one thing
2. **Test after changes** - Run tests after each modification
3. **Update fix plan** - Mark items complete as you work
4. **Minimal changes** - Smallest fix that solves the problem

## Build & Test Commands

```bash
# Install dependencies
npm install  # or pip install -r requirements.txt

# Run tests
npm test     # or pytest

# Build
npm run build  # or make build
```

## Exit Criteria

The project is complete when:

- [ ] All fix_plan.md items marked complete
- [ ] All tests passing
- [ ] No errors in build/test output
- [ ] Core requirements implemented

## Status Reporting

End every response with:

```
## Status Report

STATUS: IN_PROGRESS | COMPLETE | BLOCKED
EXIT_SIGNAL: false | true
TASKS_COMPLETED: [what you finished]
FILES_MODIFIED: [changed files]
TESTS: [pass/fail]
NEXT: [next action]
```
