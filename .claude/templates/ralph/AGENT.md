# Agent Build Instructions

> Specifications for building and running this project.

## Environment Setup

```bash
# Required tools
- Node.js >= 18 (or Python >= 3.10, Go >= 1.21, etc.)
- npm/yarn/pnpm (or pip, go mod)

# Install dependencies
npm install
```

## Build Commands

```bash
# Development build
npm run build

# Production build
npm run build:prod

# Watch mode
npm run dev
```

## Test Commands

```bash
# Run all tests
npm test

# Run specific test file
npm test -- path/to/test.ts

# Run with coverage
npm run test:coverage

# Watch mode
npm run test:watch
```

## Lint & Format

```bash
# Lint
npm run lint

# Format
npm run format

# Fix issues
npm run lint:fix
```

## Common Tasks

### Adding a new feature

1. Create feature file in `src/`
2. Add tests in `tests/`
3. Update exports if needed
4. Run tests to verify

### Fixing a bug

1. Write a failing test that reproduces the bug
2. Fix the code
3. Verify test passes
4. Check no regressions

### Running locally

```bash
npm run dev
# or
npm start
```

## Troubleshooting

### Tests failing

- Check test output for specific errors
- Verify dependencies are installed
- Check for environment issues

### Build errors

- Clear cache: `rm -rf node_modules && npm install`
- Check TypeScript errors: `npm run typecheck`
- Verify imports are correct

## Notes

- Always run tests before marking task complete
- Update fix_plan.md as you work
- Report blockers clearly in status
