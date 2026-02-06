# Deploy Readiness Check

Run a comprehensive deployment readiness check before shipping to production.

## Pre-Flight Checklist

1. **Test Suite**
   - Run the full test suite: `npm test` or `pytest` or `go test ./...`
   - All tests must pass
   - Check test coverage if available

2. **Type Checking**
   - TypeScript: `npm run typecheck` or `npx tsc --noEmit`
   - Python: `mypy .` if configured
   - Verify no type errors

3. **Linting**
   - Run linters: `npm run lint` or `ruff check .`
   - All lint rules must pass
   - No auto-fixable issues remaining

4. **Build Verification**
   - Run production build: `npm run build`
   - Verify build completes without errors
   - Check build output size is reasonable

5. **Security Scan**
   - Check for known vulnerabilities: `npm audit` or `pip audit`
   - Review any high/critical findings
   - Verify no hardcoded secrets

6. **Code Quality**
   - No TODO/FIXME comments in critical paths
   - No console.log/print debug statements
   - No commented-out code blocks

7. **Documentation**
   - README is up to date
   - API documentation reflects current endpoints
   - Environment variables are documented

8. **Database**
   - All migrations are committed
   - No pending schema changes
   - Rollback strategy documented

9. **Dependencies**
   - Lock file is committed and up to date
   - No outdated critical dependencies
   - License compliance verified

## Output Format

```
## Deployment Readiness Report

### Overall Status: [READY | NOT READY | NEEDS REVIEW]

### Checks

| Check | Status | Notes |
|-------|--------|-------|
| Tests | Pass/Fail | X tests, Y% coverage |
| Types | Pass/Fail | X errors |
| Lint | Pass/Fail | X warnings |
| Build | Pass/Fail | Size: X MB |
| Security | Pass/Fail | X vulnerabilities |
| Code Quality | Pass/Fail | X issues |

### Blockers
- [List any items that MUST be fixed before deploy]

### Warnings
- [List any items that SHOULD be addressed]

### Recommendations
- [List any nice-to-have improvements]
```

Run all available checks and produce a comprehensive report. Be thorough but practical - distinguish between blockers and nice-to-haves.
